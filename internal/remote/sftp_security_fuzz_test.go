package remote

import (
	"encoding/base64"
	"strings"
	"testing"
)

func FuzzSFTPCommandSerialization(f *testing.F) {
	for _, seed := range []string{
		"/",
		"folder/file.txt",
		"file with spaces.txt",
		"-leading-option-like-name",
		`glob[*]?quote"slash\\end`,
		"line\nbreak",
		"carriage\rreturn",
		"nul\x00byte",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		quoted := sftpQuote(input)
		if len(quoted) < 2 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
			t.Fatalf("SFTP operand is not bounded by double quotes: %q", quoted)
		}
		if strings.HasPrefix(input, "-") && !strings.HasPrefix(quoted, `"./`) {
			t.Fatalf("option-like SFTP operand was not neutralized: input=%q quoted=%q", input, quoted)
		}
		// Escaping adds at most one byte per input byte plus the two quotes and
		// the optional ./ prefix used to neutralize a leading dash.
		if len(quoted) > 2*len(input)+4 {
			t.Fatalf("SFTP quoting expanded unexpectedly: input=%d quoted=%d", len(input), len(quoted))
		}

		command := "get " + quoted
		stream, err := buildSFTPCommandStream([]string{command})
		unsafe := strings.ContainsAny(input, "\x00\r\n")
		if unsafe {
			if err == nil {
				t.Fatalf("SFTP command stream accepted a control character: input=%q stream=%q", input, stream)
			}
			if stream != "" {
				t.Fatalf("rejected SFTP command returned a partial stream: %q", stream)
			}
			return
		}
		if err != nil {
			t.Fatalf("safe SFTP operand was rejected: input=%q err=%v", input, err)
		}
		want := command + "\nquit\n"
		if stream != want {
			t.Fatalf("SFTP command stream changed serialized command: got=%q want=%q", stream, want)
		}
	})
}

func FuzzSFTPFingerprintParser(f *testing.F) {
	for _, seed := range []string{
		"example.test ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAABAgMEBQYHCAkKCwwNDg8QERITFBUWFxgZGhscHR4f",
		"example.test ssh-rsa AAAAC3NzaC1lZDI1NTE5AAAAIAABAgMEBQYHCAkKCwwNDg8QERITFBUWFxgZGhscHR4f",
		"example.test ssh-ed25519 not-base64",
		"",
		"host unsupported AAAA",
		"host ssh-ed25519 AAAA",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		fingerprint, err := fingerprintScannedKey(line)
		if err != nil {
			if fingerprint != "" {
				t.Fatalf("failed fingerprint parse returned data: %q", fingerprint)
			}
			return
		}

		const prefix = "SHA256:"
		if !strings.HasPrefix(fingerprint, prefix) {
			t.Fatalf("successful fingerprint lacks SHA256 prefix: %q", fingerprint)
		}
		digest, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(fingerprint, prefix))
		if decodeErr != nil || len(digest) != 32 {
			t.Fatalf("successful fingerprint is not a SHA-256 digest: fingerprint=%q len=%d err=%v", fingerprint, len(digest), decodeErr)
		}
		if scanKeyAlgorithm(line) == "" {
			t.Fatalf("fingerprint parser accepted a line without a supported algorithm: %q", line)
		}
		again, againErr := fingerprintScannedKey(line)
		if againErr != nil || again != fingerprint {
			t.Fatalf("fingerprint parser is not deterministic: first=%q second=%q err=%v", fingerprint, again, againErr)
		}
	})
}
