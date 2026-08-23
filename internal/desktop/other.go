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

	"github.com/bren-wp/by-ftp/internal/api"
	"github.com/bren-wp/by-ftp/internal/i18n"
	"github.com/bren-wp/by-ftp/internal/model"
	"github.com/bren-wp/by-ftp/internal/usererror"
)

func prompt(reader *bufio.Reader, label, fallback string) (string, error) {
	if fallback != "" {
		fmt.Printf("%s [%s]: ", label, fallback)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback, nil
	}
	return line, nil
}

func stty(args ...string) error {
	for _, candidate := range []string{"/bin/stty", "/usr/bin/stty", "stty"} {
		cmd := exec.Command(candidate, args...)
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return errors.New("terminal credential echo could not be disabled")
}

func promptSecret(reader *bufio.Reader, language, label string) (string, error) {
	fmt.Printf("%s: ", label)
	if err := stty("-echo"); err != nil {
		fmt.Println()
		return "", errors.New(i18n.T(language, "terminal.secret_hide_failed"))
	}
	defer func() { _ = stty("echo"); fmt.Println() }()
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func defaultPort(protocol string) int {
	switch protocol {
	case "sftp":
		return 22
	case "ftpsi":
		return 990
	default:
		return 21
	}
}

func printRemoteItems(language string, items []model.Item) {
	for _, item := range items {
		kind := i18n.T(language, "terminal.item_file")
		if item.IsDirectory {
			kind = i18n.T(language, "terminal.item_folder")
		} else if item.IsSymlink {
			kind = i18n.T(language, "terminal.item_link")
		}
		fmt.Printf("%-8s %12d  %s\n", kind, item.Size, item.Name)
	}
	fmt.Println(i18n.T(language, "terminal.items", len(items)))
}

func terminalStatus(status string) bool {
	switch status {
	case "done", "failed", "cancelled", "skipped":
		return true
	default:
		return false
	}
}

func terminalTransferResult(language string, job model.TransferJob) (bool, error) {
	if !terminalStatus(job.Status) {
		return false, nil
	}
	switch job.Status {
	case "done":
		fmt.Println(i18n.T(language, "terminal.transfer_done"))
		return true, nil
	case "skipped":
		return true, errors.New(i18n.T(language, "terminal.transfer_skipped"))
	case "cancelled":
		return true, context.Canceled
	default:
		if strings.TrimSpace(job.Error) != "" {
			return true, errors.New(job.Error)
		}
		return true, errors.New(i18n.T(language, "terminal.transfer_failed"))
	}
}

func waitTransfer(engine *api.Engine, language, jobID string) error {
	seen := false
	for {
		jobs := engine.Transfers()
		found := false
		for _, job := range jobs {
			if job.ID != jobID {
				continue
			}
			found = true
			seen = true
			if done, err := terminalTransferResult(language, job); done {
				return err
			}
			break
		}
		if seen && !found {
			return errors.New(i18n.T(language, "terminal.transfer_missing"))
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func connectTerminal(engine *api.Engine, reader *bufio.Reader, language string) (model.ConnectionConfig, error) {
	protocol, err := prompt(reader, i18n.T(language, "terminal.protocol"), "sftp")
	if err != nil {
		return model.ConnectionConfig{}, err
	}
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	host, err := prompt(reader, i18n.T(language, "terminal.server"), "")
	if err != nil {
		return model.ConnectionConfig{}, err
	}
	portText, err := prompt(reader, i18n.T(language, "terminal.port"), strconv.Itoa(defaultPort(protocol)))
	if err != nil {
		return model.ConnectionConfig{}, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return model.ConnectionConfig{}, errors.New(i18n.T(language, "terminal.port_number"))
	}
	username, err := prompt(reader, i18n.T(language, "terminal.username"), "")
	if err != nil {
		return model.ConnectionConfig{}, err
	}
	cfg := model.ConnectionConfig{Protocol: protocol, Host: host, Port: port, Username: username}
	if protocol == "sftp" {
		keyPath, err := prompt(reader, i18n.T(language, "terminal.private_key"), "")
		if err != nil {
			return model.ConnectionConfig{}, err
		}
		cfg.PrivateKeyPath = strings.TrimSpace(keyPath)
		if cfg.PrivateKeyPath == "" {
			return model.ConnectionConfig{}, errors.New(i18n.T(language, "terminal.sftp_key_required"))
		}
		passphrase, err := promptSecret(reader, language, i18n.T(language, "terminal.key_passphrase"))
		if err != nil {
			return model.ConnectionConfig{}, err
		}
		if passphrase != "" {
			return model.ConnectionConfig{}, errors.New(i18n.T(language, "terminal.sftp_passphrase_unsupported"))
		}
	} else {
		password, err := promptSecret(reader, language, i18n.T(language, "terminal.password"))
		if err != nil {
			return model.ConnectionConfig{}, err
		}
		cfg.Password = password
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	result, err := engine.Connect(ctx, "", cfg, "", false)
	if err != nil {
		return model.ConnectionConfig{}, err
	}
	if result.RequiresTrust {
		fmt.Printf("\n%s\n%s\n", i18n.T(language, "terminal.fingerprint"), result.Fingerprint)
		answer, err := prompt(reader, i18n.T(language, "terminal.trust"), "no")
		if err != nil {
			engine.CancelPendingTrust()
			return model.ConnectionConfig{}, err
		}
		if !i18n.IsAffirmative(language, answer) {
			engine.CancelPendingTrust()
			return model.ConnectionConfig{}, context.Canceled
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), 75*time.Second)
		defer cancel2()
		result, err = engine.Connect(ctx2, "", cfg, result.Fingerprint, false)
		if err != nil {
			return model.ConnectionConfig{}, err
		}
	}
	if !result.Connected {
		return model.ConnectionConfig{}, errors.New(i18n.T(language, "terminal.server_not_connected"))
	}
	cfg.Password, cfg.Passphrase = "", ""
	return cfg, nil
}

func terminalLanguage(engine *api.Engine) (model.Settings, string) {
	settings, err := engine.Settings()
	if err != nil {
		return model.Settings{Language: i18n.DefaultLanguage}, i18n.DefaultLanguage
	}
	settings.Language = i18n.Normalize(settings.Language)
	return settings, settings.Language
}

func setTerminalLanguage(engine *api.Engine, settings model.Settings, code string) (model.Settings, string, error) {
	if !i18n.IsSupported(code) {
		return settings, i18n.Normalize(settings.Language), errors.New("unsupported language")
	}
	settings.Language = i18n.Normalize(code)
	saved, err := engine.SetSettings(settings)
	if err != nil {
		return settings, i18n.Normalize(settings.Language), err
	}
	return saved, i18n.Normalize(saved.Language), nil
}

func usage(language, syntax string) string {
	prefix := map[string]string{
		"en": "Usage", "hr": "Upotreba", "de": "Verwendung", "fr": "Utilisation", "es": "Uso", "tr": "Kullanım",
		"el": "Χρήση", "pt": "Utilização", "zh": "用法", "ru": "Использование", "hi": "उपयोग", "ja": "使用法",
	}[i18n.Normalize(language)]
	return prefix + ": " + syntax
}

func runTerminal(engine *api.Engine, version string) error {
	reader := bufio.NewReader(os.Stdin)
	settings, language := terminalLanguage(engine)
	fmt.Println(i18n.T(language, "terminal.title", version, runtime.GOOS))
	fmt.Println(i18n.T(language, "terminal.privacy"))
	cfg, err := connectTerminal(engine, reader, language)
	if err != nil {
		return errors.New(usererror.MessageFor(language, err, i18n.T(language, "terminal.connect_failed")))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = engine.Disconnect(ctx)
	}()
	current := "/"
	if cfg.Protocol == "sftp" {
		current = "."
	}
	fmt.Println(i18n.T(language, "terminal.connected", cfg.Host, cfg.Port))
	fmt.Println(i18n.T(language, "terminal.commands"))
	fmt.Println(i18n.T(language, "terminal.quote_paths"))
	for {
		fmt.Printf("byftp:%s> ", current)
		line, readErr := reader.ReadString('\n')
		if readErr != nil && len(line) == 0 {
			return nil
		}
		fields, parseErr := parseTerminalArgs(line)
		if parseErr != nil {
			fmt.Println(i18n.T(language, "terminal.invalid_command", parseErr.Error()))
			continue
		}
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "quit", "exit":
			if len(fields) != 1 {
				fmt.Println(usage(language, "quit"))
				continue
			}
			return nil
		case "help":
			if len(fields) != 1 {
				fmt.Println(usage(language, "help"))
				continue
			}
			fmt.Println(i18n.T(language, "terminal.commands"))
			fmt.Println(i18n.T(language, "terminal.quote_paths"))
		case "language":
			if len(fields) != 2 || !i18n.IsSupported(fields[1]) {
				fmt.Println(i18n.T(language, "terminal.language_usage"))
				continue
			}
			next, nextLanguage, setErr := setTerminalLanguage(engine, settings, fields[1])
			if setErr != nil {
				fmt.Println(usererror.MessageFor(language, setErr, i18n.T(language, "settings.save_failed_body")))
				continue
			}
			settings, language = next, nextLanguage
			fmt.Println(i18n.T(language, "terminal.language_saved", i18n.LanguageByCode(language).NativeName))
		case "pwd":
			if len(fields) != 1 {
				fmt.Println(usage(language, "pwd"))
				continue
			}
			fmt.Println(current)
		case "ls":
			if len(fields) > 2 {
				fmt.Println(usage(language, "ls [path]"))
				continue
			}
			target := current
			if len(fields) > 1 {
				target = fields[1]
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			items, e := engine.RemoteList(ctx, target)
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
				continue
			}
			printRemoteItems(language, items)
		case "cd":
			if len(fields) != 2 {
				fmt.Println(usage(language, "cd <path>"))
				continue
			}
			target := fields[1]
			if !strings.HasPrefix(target, "/") && target != "." {
				target = path.Join(current, target)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			items, e := engine.RemoteList(ctx, target)
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
				continue
			}
			current = target
			printRemoteItems(language, items)
		case "mkdir":
			if len(fields) != 2 {
				fmt.Println(usage(language, "mkdir <name>"))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			e := engine.RemoteMkdir(ctx, current, fields[1])
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
			}
		case "rename":
			if len(fields) != 3 {
				fmt.Println(usage(language, "rename <old> <new>"))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			e := engine.RemoteRename(ctx, current, fields[1], fields[2])
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
			}
		case "delete":
			if len(fields) != 2 {
				fmt.Println(usage(language, "delete <name>"))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			items, e := engine.RemoteList(ctx, current)
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
				continue
			}
			var found *model.Item
			for i := range items {
				if items[i].Name == fields[1] {
					item := items[i]
					found = &item
					break
				}
			}
			if found == nil {
				fmt.Println(i18n.T(language, "error.not_found"))
				continue
			}
			ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
			e = engine.RemoteDelete(ctx, current, found.Name, found.IsDirectory)
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
			}
		case "chmod":
			if len(fields) != 3 {
				fmt.Println(usage(language, "chmod <mode> <name>"))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			e := engine.RemoteChmod(ctx, current, fields[2], fields[1])
			cancel()
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
			}
		case "get", "put":
			if len(fields) != 3 {
				fmt.Println(usage(language, fields[0]+" <source> <destination>"))
				continue
			}
			direction := strings.ToLower(fields[0])
			localArg, remoteArg := fields[2], fields[1]
			if direction == "put" {
				direction, localArg, remoteArg = "upload", fields[1], fields[2]
			} else {
				direction = "download"
			}
			localPath, e := filepath.Abs(localArg)
			if e != nil {
				fmt.Println(i18n.T(language, "error.invalid_name"))
				continue
			}
			if !strings.HasPrefix(remoteArg, "/") {
				remoteArg = path.Join(current, remoteArg)
			}
			job, e := engine.AddTransfer(direction, localPath, remoteArg, filepath.Dir(localPath))
			if e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "error.generic")))
				continue
			}
			if e = waitTransfer(engine, language, job.ID); e != nil {
				fmt.Println(usererror.MessageFor(language, e, i18n.T(language, "terminal.transfer_failed")))
			}
		default:
			fmt.Println(i18n.T(language, "terminal.unknown_command"))
		}
	}
}

func Run(engine *api.Engine, version string) error { return runTerminal(engine, version) }
