//go:build !windows

package desktop

import (
	"errors"
	"strings"
)

const (
	maxTerminalCommandLength = 32 << 10
	maxTerminalArguments     = 64
)

// parseTerminalArgs parsira interaktivnu naredbu bez pokretanja ljuske.
// Navodnici služe samo grupiranju argumenata s razmacima; rezultat se i dalje
// prosljeđuje isključivo tipiziranim ByFTP operacijama, nikada shellu.
func parseTerminalArgs(line string) ([]string, error) {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if len(line) > maxTerminalCommandLength {
		return nil, errors.New("naredba je preduga")
	}
	if strings.ContainsAny(line, "\x00\r\n") {
		return nil, errors.New("naredba sadrži nedopuštene kontrolne znakove")
	}

	args := make([]string, 0, 4)
	var token strings.Builder
	var quote byte
	tokenStarted := false

	appendToken := func() error {
		args = append(args, token.String())
		token.Reset()
		tokenStarted = false
		if len(args) > maxTerminalArguments {
			return errors.New("naredba ima previše argumenata")
		}
		return nil
	}

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
				tokenStarted = true
				continue
			}
			if ch == '\\' && i+1 < len(line) && (line[i+1] == quote || line[i+1] == '\\') {
				i++
				token.WriteByte(line[i])
				tokenStarted = true
				continue
			}
			token.WriteByte(ch)
			tokenStarted = true
			continue
		}

		switch ch {
		case '\'', '"':
			quote = ch
			tokenStarted = true
		case ' ', '\t':
			if tokenStarted {
				if err := appendToken(); err != nil {
					return nil, err
				}
			}
		default:
			token.WriteByte(ch)
			tokenStarted = true
		}
	}
	if quote != 0 {
		return nil, errors.New("navodnik u naredbi nije zatvoren")
	}
	if tokenStarted {
		if err := appendToken(); err != nil {
			return nil, err
		}
	}
	return args, nil
}
