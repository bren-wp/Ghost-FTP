//go:build linux

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

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
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
	line = strings.TrimRight(line, "\r\n")
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

// connectTerminal deliberately uses the same ConnectionConfig model and Engine
// as Windows. SFTP accepts either password authentication or a private key with
// an optional passphrase; the Linux frontend must not impose a stricter,
// incompatible authentication subset on top of the shared engine.
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
			password, err := promptSecret(reader, language, i18n.T(language, "terminal.password"))
			if err != nil {
				return model.ConnectionConfig{}, err
			}
			cfg.Password = password
		} else {
			passphrase, err := promptSecret(reader, language, i18n.T(language, "terminal.key_passphrase"))
			if err != nil {
				return model.ConnectionConfig{}, err
			}
			cfg.Passphrase = passphrase
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
		cfg.Password, cfg.Passphrase = "", ""
		return model.ConnectionConfig{}, err
	}
	if result.RequiresTrust {
		fmt.Printf("\n%s\n%s\n", i18n.T(language, "terminal.fingerprint"), result.Fingerprint)
		answer, err := prompt(reader, i18n.T(language, "terminal.trust"), "no")
		if err != nil {
			engine.CancelPendingTrust()
			cfg.Password, cfg.Passphrase = "", ""
			return model.ConnectionConfig{}, err
		}
		if !i18n.IsAffirmative(language, answer) {
			engine.CancelPendingTrust()
			cfg.Password, cfg.Passphrase = "", ""
			return model.ConnectionConfig{}, context.Canceled
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), 75*time.Second)
		defer cancel2()
		result, err = engine.Connect(ctx2, "", cfg, result.Fingerprint, false)
		if err != nil {
			cfg.Password, cfg.Passphrase = "", ""
			return model.ConnectionConfig{}, err
		}
	}
	if !result.Connected {
		cfg.Password, cfg.Passphrase = "", ""
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

func printTerminalSettings(settings model.Settings) {
	fmt.Printf("language=%s\n", settings.Language)
	fmt.Printf("parallelism=%d\n", settings.Parallelism)
	fmt.Printf("conflict=%s\n", settings.ConflictPolicy)
	fmt.Printf("retries=%d\n", settings.AutoRetryCount)
	fmt.Printf("retry-delay=%d\n", settings.RetryDelaySeconds)
	fmt.Printf("timeout=%d\n", settings.ConnectionTimeoutSeconds)
	fmt.Printf("confirm-delete=%t\n", settings.ConfirmDelete)
}

func parseTerminalBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errors.New("expected true/false, yes/no, on/off or 1/0")
	}
}

func updateTerminalSetting(settings model.Settings, option, value string) (model.Settings, error) {
	next := settings
	var err error
	switch strings.ToLower(option) {
	case "parallelism":
		next.Parallelism, err = strconv.Atoi(value)
	case "conflict":
		next.ConflictPolicy = strings.ToLower(strings.TrimSpace(value))
	case "retries":
		next.AutoRetryCount, err = strconv.Atoi(value)
	case "retry-delay":
		next.RetryDelaySeconds, err = strconv.Atoi(value)
	case "timeout":
		next.ConnectionTimeoutSeconds, err = strconv.Atoi(value)
	case "confirm-delete":
		next.ConfirmDelete, err = parseTerminalBool(value)
	default:
		return settings, errors.New("unknown setting")
	}
	if err != nil {
		return settings, err
	}
	return next, nil
}

func printTransferJobs(engine *api.Engine) {
	jobs := engine.Transfers()
	if len(jobs) == 0 {
		fmt.Println("jobs=0")
		return
	}
	for _, job := range jobs {
		fmt.Printf("%s  %-9s %-10s %6.1f%%  %s -> %s\n", job.ID, job.Direction, job.Status, job.Progress*100, job.LocalPath, job.RemotePath)
	}
}

func printProfiles(engine *api.Engine) error {
	profiles, err := engine.Profiles()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		fmt.Println("profiles=0")
		return nil
	}
	for _, profile := range profiles {
		fmt.Printf("%s  %s  %s://%s:%d  %s\n", profile.ID, profile.Name, profile.Protocol, profile.Host, profile.Port, profile.Username)
	}
	return nil
}

func usage(language, syntax string) string {
	prefix := map[string]string{
		"en": "Usage", "hr": "Upotreba", "de": "Verwendung", "fr": "Utilisation", "es": "Uso", "tr": "Kullanım",
		"el": "Χρήση", "pt": "Utilização", "zh": "用法", "ru": "Использование", "hi": "उपयोग", "ja": "使用法",
	}[i18n.Normalize(language)]
	if prefix == "" {
		prefix = "Usage"
	}
	return prefix + ": " + syntax
}

func runTerminal(engine *api.Engine, version string) error {
	reader := bufio.NewReader(os.Stdin)
	settings, language := terminalLanguage(engine)
	fmt.Println("────────────────────────────────────────────────────────")
	fmt.Println(i18n.T(language, "terminal.title", version, runtime.GOOS))
	fmt.Println(i18n.T(language, "terminal.privacy"))
	fmt.Println("────────────────────────────────────────────────────────")
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
	fmt.Println("jobs | pause | resume | cancel <id> | retry <id> | clear | profiles | settings | set <option> <value>")
	for {
		fmt.Printf("GhostFTP:%s> ", current)
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
			fmt.Println("jobs | pause | resume | cancel <id> | retry <id> | clear | profiles | settings | set <option> <value>")
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
		case "settings":
			if len(fields) != 1 {
				fmt.Println(usage(language, "settings"))
				continue
			}
			currentSettings, settingsErr := engine.Settings()
			if settingsErr != nil {
				fmt.Println(usererror.MessageFor(language, settingsErr, i18n.T(language, "error.generic")))
				continue
			}
			settings = currentSettings
			printTerminalSettings(settings)
		case "set":
			if len(fields) != 3 {
				fmt.Println(usage(language, "set <parallelism|conflict|retries|retry-delay|timeout|confirm-delete> <value>"))
				continue
			}
			next, updateErr := updateTerminalSetting(settings, fields[1], fields[2])
			if updateErr != nil {
				fmt.Println(updateErr)
				continue
			}
			saved, saveErr := engine.SetSettings(next)
			if saveErr != nil {
				fmt.Println(usererror.MessageFor(language, saveErr, i18n.T(language, "settings.save_failed_body")))
				continue
			}
			settings = saved
			fmt.Println("saved")
		case "profiles":
			if len(fields) != 1 {
				fmt.Println(usage(language, "profiles"))
				continue
			}
			if profileErr := printProfiles(engine); profileErr != nil {
				fmt.Println(usererror.MessageFor(language, profileErr, i18n.T(language, "error.generic")))
			}
		case "jobs":
			if len(fields) != 1 {
				fmt.Println(usage(language, "jobs"))
				continue
			}
			printTransferJobs(engine)
		case "pause":
			if len(fields) != 1 {
				fmt.Println(usage(language, "pause"))
				continue
			}
			engine.PauseTransfers()
			fmt.Println("paused")
		case "resume":
			if len(fields) != 1 {
				fmt.Println(usage(language, "resume"))
				continue
			}
			engine.ResumeTransfers()
			fmt.Println("resumed")
		case "cancel":
			if len(fields) != 2 {
				fmt.Println(usage(language, "cancel <id>"))
				continue
			}
			if cancelErr := engine.CancelTransfer(fields[1]); cancelErr != nil {
				fmt.Println(usererror.MessageFor(language, cancelErr, i18n.T(language, "error.generic")))
			}
		case "retry":
			if len(fields) != 2 {
				fmt.Println(usage(language, "retry <id>"))
				continue
			}
			if retryErr := engine.RetryTransfer(fields[1]); retryErr != nil {
				fmt.Println(usererror.MessageFor(language, retryErr, i18n.T(language, "error.generic")))
			}
		case "clear":
			if len(fields) != 1 {
				fmt.Println(usage(language, "clear"))
				continue
			}
			engine.ClearFinishedTransfers()
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
