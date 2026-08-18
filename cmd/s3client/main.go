package main

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
	"strings"
	"time"

	"brendigo.com/byftp/internal/clientmode"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/s3client"
)

var version = "dev"

func prompt(reader *bufio.Reader, label, fallback string) (string, error) {
	if fallback != "" { fmt.Printf("%s [%s]: ", label, fallback) } else { fmt.Printf("%s: ", label) }
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 { return "", err }
	line = strings.TrimSpace(line)
	if line == "" { return fallback, nil }
	return line, nil
}

func promptSecret(reader *bufio.Reader, label string) (string, error) {
	if runtime.GOOS == "windows" {
		value, ok := platform.PromptDialog("ByFTP S3 Client", label, "")
		if !ok { return "", context.Canceled }
		return value, nil
	}
	fmt.Printf("%s: ", label)
	var stty *exec.Cmd
	for _, candidate := range []string{"/bin/stty", "/usr/bin/stty", "stty"} {
		stty = exec.Command(candidate, "-echo")
		stty.Stdin = os.Stdin
		if stty.Run() == nil { break }
		stty = nil
	}
	if stty == nil { fmt.Println(); return "", errors.New("nije moguće sigurno sakriti unos S3 tajne") }
	defer func() {
		for _, candidate := range []string{"/bin/stty", "/usr/bin/stty", "stty"} {
			cmd := exec.Command(candidate, "echo"); cmd.Stdin = os.Stdin
			if cmd.Run() == nil { break }
		}
		fmt.Println()
	}()
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 { return "", err }
	return strings.TrimRight(line, "\r\n"), nil
}

func safePrefix(current, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" || arg == "." { return current, nil }
	var p string
	if strings.HasPrefix(arg, "/") { p = strings.TrimPrefix(arg, "/") } else { p = path.Join(current, arg) }
	p = path.Clean("/" + p)
	if p == "/" { return "", nil }
	if strings.ContainsAny(p, "\x00\r\n") { return "", errors.New("S3 putanja sadrži nedopuštene znakove") }
	return strings.TrimPrefix(p, "/"), nil
}

func keyAt(current, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "\x00\r\n") {
		return "", errors.New("neispravan naziv S3 objekta")
	}
	if strings.HasPrefix(name, "/") { return strings.TrimPrefix(path.Clean(name), "/"), nil }
	if current == "" { return name, nil }
	return path.Join(current, name), nil
}

func main() {
	platform.HardenProcessPrivacy()
	mode := clientmode.S3
	release, ok := platform.AcquireSingleInstance(mode.InstanceKey())
	if !ok { fmt.Fprintln(os.Stderr, mode.ProductName()+" je već pokrenut."); return }
	defer release()

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("ByFTP S3 Client %s\n", version)
	fmt.Println("S3/object-storage klijent. Secret key ostaje samo u memoriji procesa.")
	endpoint, err := prompt(reader, "S3 endpoint", "https://s3.amazonaws.com"); if err != nil { fmt.Fprintln(os.Stderr, err); return }
	region, err := prompt(reader, "Regija", "us-east-1"); if err != nil { fmt.Fprintln(os.Stderr, err); return }
	bucket, err := prompt(reader, "Bucket", ""); if err != nil { fmt.Fprintln(os.Stderr, err); return }
	accessKey, err := prompt(reader, "Access key", ""); if err != nil { fmt.Fprintln(os.Stderr, err); return }
	secretKey, err := promptSecret(reader, "Secret key"); if err != nil { fmt.Fprintln(os.Stderr, err); return }
	client, err := s3client.New(s3client.Config{Endpoint: endpoint, Region: region, Bucket: bucket, AccessKey: accessKey, SecretKey: secretKey})
	secretKey = ""
	if err != nil { fmt.Fprintln(os.Stderr, err); return }
	current := ""
	fmt.Println("Naredbe: ls, cd, pwd, put, get, mkdir, rename, delete, help, quit")
	for {
		display := "/" + current
		fmt.Printf("s3:%s> ", display)
		line, readErr := reader.ReadString('\n')
		if readErr != nil && len(line) == 0 { return }
		fields := strings.Fields(strings.TrimSpace(line)); if len(fields) == 0 { continue }
		switch strings.ToLower(fields[0]) {
		case "quit", "exit": return
		case "help": fmt.Println("ls [prefix] | cd <prefix> | pwd | put <local> <remote> | get <remote> <local> | mkdir <prefix> | rename <staro> <novo> | delete <objekt> | quit")
		case "pwd": fmt.Println("/" + current)
		case "ls":
			target := current
			if len(fields) > 1 { target, err = safePrefix(current, fields[1]); if err != nil { fmt.Println(err); continue } }
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); items, e := client.List(ctx, target); cancel()
			if e != nil { fmt.Println("Listanje nije uspjelo:", e); continue }
			for _, item := range items { kind := "OBJ"; if item.Prefix { kind = "DIR" }; fmt.Printf("%-3s %12d  %s\n", kind, item.Size, item.Name) }
		case "cd":
			if len(fields) != 2 { fmt.Println("Upotreba: cd <prefix>"); continue }
			target, e := safePrefix(current, fields[1]); if e != nil { fmt.Println(e); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); _, e = client.List(ctx, target); cancel()
			if e != nil { fmt.Println("Prefix nije moguće otvoriti:", e); continue }; current = target
		case "put":
			if len(fields) != 3 { fmt.Println("Upotreba: put <local> <remote>"); continue }
			local, e := filepath.Abs(fields[1]); if e != nil { fmt.Println("Neispravna lokalna putanja."); continue }
			key, e := keyAt(current, fields[2]); if e != nil { fmt.Println(e); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour); e = client.Put(ctx, local, key); cancel(); if e != nil { fmt.Println("Upload nije uspio:", e) } else { fmt.Println("Upload dovršen.") }
		case "get":
			if len(fields) != 3 { fmt.Println("Upotreba: get <remote> <local>"); continue }
			key, e := keyAt(current, fields[1]); if e != nil { fmt.Println(e); continue }
			local, e := filepath.Abs(fields[2]); if e != nil { fmt.Println("Neispravna lokalna putanja."); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour); e = client.Get(ctx, key, local); cancel(); if e != nil { fmt.Println("Download nije uspio:", e) } else { fmt.Println("Download dovršen.") }
		case "mkdir":
			if len(fields) != 2 { fmt.Println("Upotreba: mkdir <prefix>"); continue }
			key, e := keyAt(current, fields[1]); if e != nil { fmt.Println(e); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); e = client.Mkdir(ctx, key); cancel(); if e != nil { fmt.Println("Prefix nije izrađen:", e) }
		case "rename":
			if len(fields) != 3 { fmt.Println("Upotreba: rename <staro> <novo>"); continue }
			oldKey, e := keyAt(current, fields[1]); if e != nil { fmt.Println(e); continue }
			newKey, e := keyAt(current, fields[2]); if e != nil { fmt.Println(e); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute); e = client.Rename(ctx, oldKey, newKey); cancel(); if e != nil { fmt.Println("Preimenovanje nije uspjelo:", e) }
		case "delete":
			if len(fields) != 2 { fmt.Println("Upotreba: delete <objekt>"); continue }
			key, e := keyAt(current, fields[1]); if e != nil { fmt.Println(e); continue }
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); e = client.Delete(ctx, key); cancel(); if e != nil { fmt.Println("Brisanje nije uspjelo:", e) }
		default: fmt.Println("Nepoznata naredba. Upišite 'help'.")
		}
	}
}
