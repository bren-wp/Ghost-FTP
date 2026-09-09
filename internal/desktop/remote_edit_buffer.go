package desktop

import (
	"errors"
	"strings"
)

type remoteEditNewline uint8

const (
	remoteEditNewlineLF remoteEditNewline = iota
	remoteEditNewlineCRLF
	remoteEditNewlineCR
)

var errRemoteEditMixedNewlines = errors.New("remote text editor refuses mixed line endings to avoid changing unrelated lines")

type remoteEditBuffer struct {
	Text    string
	Newline remoteEditNewline
}

func newRemoteEditBuffer(raw string) (remoteEditBuffer, error) {
	hasCRLF := strings.Contains(raw, "\r\n")
	withoutCRLF := strings.ReplaceAll(raw, "\r\n", "")
	hasLF := strings.Contains(withoutCRLF, "\n")
	hasCR := strings.Contains(withoutCRLF, "\r")
	styles := 0
	for _, present := range []bool{hasCRLF, hasLF, hasCR} {
		if present {
			styles++
		}
	}
	if styles > 1 {
		return remoteEditBuffer{}, errRemoteEditMixedNewlines
	}

	buffer := remoteEditBuffer{Text: raw, Newline: remoteEditNewlineLF}
	switch {
	case hasCRLF:
		buffer.Newline = remoteEditNewlineCRLF
		buffer.Text = strings.ReplaceAll(raw, "\r\n", "\n")
	case hasCR:
		buffer.Newline = remoteEditNewlineCR
		buffer.Text = strings.ReplaceAll(raw, "\r", "\n")
	}
	return buffer, nil
}

func normalizeRemoteEditorText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func (b remoteEditBuffer) Encode(text string) string {
	text = normalizeRemoteEditorText(text)
	switch b.Newline {
	case remoteEditNewlineCRLF:
		return strings.ReplaceAll(text, "\n", "\r\n")
	case remoteEditNewlineCR:
		return strings.ReplaceAll(text, "\n", "\r")
	default:
		return text
	}
}
