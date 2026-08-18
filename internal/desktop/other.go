//go:build !windows

package desktop

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/usererror"
)

func prompt(reader *bufio.Reader, label, fallback string) (string, error) {
	if fallback != "" { fmt.Printf("%s [%s]: ", label, fallback) } else { fmt.Printf("%s: ", label) }
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 { return "", err }
	line = strings.TrimSpace(line)
	if line == "" { return fallback, nil }
	return line, nil
}

func stty(args ...string) error {
	for _, candidate := range []string{"/bin/stty", "/usr/bin/stty", "stty"} {
		cmd := exec.Command(candidate, args...)
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err == nil { return nil }
	}
	return errors.New("nije moguće sigurno sakriti unos vjerodajnice u terminalu")
}

func promptSecret(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	if err := stty("-echo"); err != nil { fmt.Println(); return "", err }
	defer func() { _ = stty("echo"); fmt.Println() }()
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 { return "", err }
	return strings.TrimRight(line, "\r\n"), nil
}

func defaultPort(protocol string) int {
	switch protocol { case "sftp": return 22; case "ftpsi": return 990; default: return 21 }
}

func printRemoteItems(items []model.Item) {
	for _, item := range items {
		kind := "dat"
		if item.IsDirectory { kind = "map" } else if item.IsSymlink { kind = "link" }
		fmt.Printf("%-4s %12d  %s\n", kind, item.Size, item.Name)
	}
	fmt.Printf("%d stavki\n", len(items))
}

func terminalStatus(status string) bool {
	switch status { case "done", "failed", "cancelled", "skipped": return true; default: return false }
}

func waitTransfer(engine *api.Engine, jobID string) error {
	seq := int64(0)
	for {
		events, next := engine.TransferEvents(seq); seq = next
		for _, event := range events {
			if event.Job == nil || event.Job.ID != jobID { continue }
			job := *event.Job
			if !terminalStatus(job.Status) { continue }
			switch job.Status {
			case "done": fmt.Println("Prijenos dovršen."); return nil
			case "skipped": return errors.New("prijenos je preskočen jer odredište već postoji")
			case "cancelled": return context.Canceled
			default:
				if strings.TrimSpace(job.Error) != "" { return errors.New(job.Error) }
				return errors.New("prijenos nije uspio")
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func connectTerminal(engine *api.Engine, reader *bufio.Reader) (model.ConnectionConfig, error) {
	protocol, err := prompt(reader, "Protokol (ftp/ftps/ftpsi/sftp)", "sftp"); if err != nil { return model.ConnectionConfig{}, err }
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	host, err := prompt(reader, "Poslužitelj", ""); if err != nil { return model.ConnectionConfig{}, err }
	portText, err := prompt(reader, "Port", strconv.Itoa(defaultPort(protocol))); if err != nil { return model.ConnectionConfig{}, err }
	port, err := strconv.Atoi(portText); if err != nil { return model.ConnectionConfig{}, errors.New("port mora biti broj") }
	username, err := prompt(reader, "Korisničko ime", ""); if err != nil { return model.ConnectionConfig{}, err }
	cfg := model.ConnectionConfig{Protocol: protocol, Host: host, Port: port, Username: username}
	if protocol == "sftp" {
		keyPath, err := prompt(reader, "Privatni ključ", ""); if err != nil { return model.ConnectionConfig{}, err }
		cfg.PrivateKeyPath = strings.TrimSpace(keyPath)
		if cfg.PrivateKeyPath == "" { return model.ConnectionConfig{}, errors.New("Linux/macOS SFTP izdanje zahtijeva eksplicitni privatni ključ") }
		passphrase, err := promptSecret(reader, "Zaporka privatnog ključa (Enter ako je nema)"); if err != nil { return model.ConnectionConfig{}, err }
		if passphrase != "" { return model.ConnectionConfig{}, errors.New("Linux/macOS SFTP izdanje trenutačno podržava privatni ključ bez passphrasea") }
	} else {
		password, err := promptSecret(reader, "Lozinka"); if err != nil { return model.ConnectionConfig{}, err }
		cfg.Password = password
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second); defer cancel()
	result, err := engine.Connect(ctx, "", cfg, "", false); if err != nil { return model.ConnectionConfig{}, err }
	if result.RequiresTrust {
		fmt.Printf("\nSFTP host-key fingerprint:\n%s\n", result.Fingerprint)
		answer, err := prompt(reader, "Vjerujete li ovom poslužitelju? (da/ne)", "ne"); if err != nil { engine.CancelPendingTrust(); return model.ConnectionConfig{}, err }
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "da" && answer != "d" && answer != "yes" && answer != "y" { engine.CancelPendingTrust(); return model.ConnectionConfig{}, context.Canceled }
		ctx2, cancel2 := context.WithTimeout(context.Background(), 75*time.Second); defer cancel2()
		result, err = engine.Connect(ctx2, "", cfg, result.Fingerprint, false); if err != nil { return model.ConnectionConfig{}, err }
	}
	if !result.Connected { return model.ConnectionConfig{}, errors.New("poslužitelj nije potvrdio aktivnu vezu") }
	cfg.Password, cfg.Passphrase = "", ""
	return cfg, nil
}

func runTerminal(engine *api.Engine, version string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("ByFTP %s — %s terminalni klijent\n", version, runtime.GOOS)
	fmt.Println("FTP/FTPS/SFTP klijent bez telemetrije. Tajne se ne ispisuju u terminal.")
	cfg, err := connectTerminal(engine, reader)
	if err != nil { return errors.New(usererror.Message(err, "Povezivanje nije uspjelo. Provjerite podatke i pokušajte ponovno.")) }
	defer func() { ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel(); _ = engine.Disconnect(ctx) }()
	current := "/"; if cfg.Protocol == "sftp" { current = "." }
	fmt.Printf("Povezano: %s:%d\n", cfg.Host, cfg.Port)
	fmt.Println("Naredbe: ls, cd, mkdir, rename, delete, chmod, get, put, pwd, help, quit")
	fmt.Println("Putanje i nazive s razmacima stavite u jednostruke ili dvostruke navodnike.")
	for {
		fmt.Printf("byftp:%s> ", current)
		line, readErr := reader.ReadString('\n'); if readErr != nil && len(line) == 0 { return nil }
		fields, parseErr := parseTerminalArgs(line)
		if parseErr != nil { fmt.Println("Neispravna naredba: " + parseErr.Error()); continue }
		if len(fields) == 0 { continue }
		switch strings.ToLower(fields[0]) {
		case "quit", "exit":
			if len(fields) != 1 { fmt.Println("Upotreba: quit"); continue }
			return nil
		case "help":
			if len(fields) != 1 { fmt.Println("Upotreba: help"); continue }
			fmt.Println("ls [putanja] | cd <putanja> | mkdir <ime> | rename <staro> <novo> | delete <ime> | chmod <mode> <ime> | get <remote> <local> | put <local> <remote> | pwd | quit")
			fmt.Println("Primjer: get \"Ugovori/račun 2026.pdf\" \"/home/korisnik/Preuzeto/račun 2026.pdf\"")
		case "pwd":
			if len(fields) != 1 { fmt.Println("Upotreba: pwd"); continue }
			fmt.Println(current)
		case "ls":
			if len(fields) > 2 { fmt.Println("Upotreba: ls [putanja]"); continue }
			target := current; if len(fields) > 1 { target = fields[1] }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); items, e := engine.RemoteList(ctx, target); cancel()
			if e != nil { fmt.Println(usererror.Message(e, "Udaljeni direktorij nije moguće pročitati.")); continue }; printRemoteItems(items)
		case "cd":
			if len(fields) != 2 { fmt.Println("Upotreba: cd <putanja>"); continue }
			target := fields[1]; if !strings.HasPrefix(target, "/") && target != "." { target = path.Join(current, target) }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); items, e := engine.RemoteList(ctx, target); cancel()
			if e != nil { fmt.Println(usererror.Message(e, "Udaljeni direktorij nije moguće otvoriti.")); continue }; current = target; printRemoteItems(items)
		case "mkdir":
			if len(fields) != 2 { fmt.Println("Upotreba: mkdir <ime>"); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); e := engine.RemoteMkdir(ctx, current, fields[1]); cancel(); if e != nil { fmt.Println(usererror.Message(e, "Mapu nije moguće izraditi.")) }
		case "rename":
			if len(fields) != 3 { fmt.Println("Upotreba: rename <staro> <novo>"); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); e := engine.RemoteRename(ctx, current, fields[1], fields[2]); cancel(); if e != nil { fmt.Println(usererror.Message(e, "Stavku nije moguće preimenovati.")) }
		case "delete":
			if len(fields) != 2 { fmt.Println("Upotreba: delete <ime>"); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); items, e := engine.RemoteList(ctx, current); cancel(); if e != nil { fmt.Println(usererror.Message(e, "Nije moguće provjeriti udaljenu stavku.")); continue }
			var found *model.Item; for i := range items { if items[i].Name == fields[1] { item := items[i]; found = &item; break } }; if found == nil { fmt.Println("Stavka nije pronađena."); continue }
			ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second); e = engine.RemoteDelete(ctx, current, found.Name, found.IsDirectory); cancel(); if e != nil { fmt.Println(usererror.Message(e, "Stavku nije moguće obrisati.")) }
		case "chmod":
			if len(fields) != 3 { fmt.Println("Upotreba: chmod <mode> <ime>"); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second); e := engine.RemoteChmod(ctx, current, fields[2], fields[1]); cancel(); if e != nil { fmt.Println(usererror.Message(e, "Dozvole nije moguće promijeniti.")) }
		case "get", "put":
			if len(fields) != 3 { fmt.Printf("Upotreba: %s <izvor> <odredište>\n", fields[0]); continue }
			direction := strings.ToLower(fields[0]); localArg, remoteArg := fields[2], fields[1]; if direction == "put" { direction, localArg, remoteArg = "upload", fields[1], fields[2] } else { direction = "download" }
			localPath, e := filepath.Abs(localArg); if e != nil { fmt.Println("Neispravna lokalna putanja."); continue }
			if !strings.HasPrefix(remoteArg, "/") { remoteArg = path.Join(current, remoteArg) }
			job, e := engine.AddTransfer(direction, localPath, remoteArg, filepath.Dir(localPath)); if e != nil { fmt.Println(usererror.Message(e, "Prijenos nije moguće pokrenuti.")); continue }
			if e = waitTransfer(engine, job.ID); e != nil { fmt.Println(usererror.Message(e, "Prijenos nije uspio.")) }
		default: fmt.Println("Nepoznata naredba. Upišite 'help'.")
		}
	}
}

func Run(engine *api.Engine, version string) error { return runTerminal(engine, version) }
