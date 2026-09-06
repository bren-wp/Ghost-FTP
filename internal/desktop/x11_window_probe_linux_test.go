//go:build linux

package desktop

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func x11QueryTree(x *x11Client, window uint32) ([]uint32, error) {
	buf := make([]byte, 8)
	buf[0] = 15
	x.order.PutUint32(buf[4:8], window)
	if err := x.request(buf); err != nil {
		return nil, err
	}
	reply, err := x.readReply()
	if err != nil {
		return nil, err
	}
	if len(reply) < 32 {
		return nil, errors.New("X11 QueryTree reply is truncated")
	}
	count := int(x.order.Uint16(reply[16:18]))
	if count < 0 || count > 4096 || len(reply) < 32+count*4 {
		return nil, errors.New("X11 QueryTree child list is invalid")
	}
	children := make([]uint32, count)
	for i := range children {
		off := 32 + i*4
		children[i] = x.order.Uint32(reply[off : off+4])
	}
	return children, nil
}

func x11GetStringProperty(x *x11Client, window, property uint32) (string, error) {
	buf := make([]byte, 24)
	buf[0] = 20
	x.order.PutUint32(buf[4:8], window)
	x.order.PutUint32(buf[8:12], property)
	x.order.PutUint32(buf[12:16], x11AtomString)
	x.order.PutUint32(buf[16:20], 0)
	x.order.PutUint32(buf[20:24], 1024)
	if err := x.request(buf); err != nil {
		return "", err
	}
	reply, err := x.readReply()
	if err != nil {
		return "", err
	}
	if len(reply) < 32 || reply[1] == 0 {
		return "", nil
	}
	if reply[1] != 8 {
		return "", errors.New("X11 string property has an unexpected format")
	}
	count := int(x.order.Uint32(reply[16:20]))
	if count < 0 || count > 4096 || len(reply) < 32+count {
		return "", errors.New("X11 string property length is invalid")
	}
	return string(reply[32 : 32+count]), nil
}

func x11FindNamedWindow(x *x11Client, root uint32, needle string, depth int) (uint32, string, error) {
	if depth < 0 {
		return 0, "", nil
	}
	children, err := x11QueryTree(x, root)
	if err != nil {
		return 0, "", err
	}
	for _, child := range children {
		name, err := x11GetStringProperty(x, child, x11AtomWMName)
		if err != nil {
			return 0, "", err
		}
		if strings.Contains(name, needle) {
			return child, name, nil
		}
		if depth > 0 {
			found, foundName, err := x11FindNamedWindow(x, child, needle, depth-1)
			if err != nil {
				return 0, "", err
			}
			if found != 0 {
				return found, foundName, nil
			}
		}
	}
	return 0, "", nil
}

// TestX11FindGhostFTPWindow verifies the mapped production application from a
// separate X11 client. The runtime workflow launches the real binary first and
// then enables this opt-in probe; no external xwininfo/xdotool package is used.
func TestX11FindGhostFTPWindow(t *testing.T) {
	if os.Getenv("GHOSTFTP_X11_FIND_WINDOW") != "1" {
		t.Skip("set GHOSTFTP_X11_FIND_WINDOW=1 while the production GUI is running")
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		x, err := connectLocalX11()
		if err != nil {
			t.Fatalf("second X11 client could not connect: %v", err)
		}
		window, name, findErr := x11FindNamedWindow(x, x.root, "Ghost FTP", 3)
		x.close()
		if findErr != nil {
			t.Fatalf("X11 production-window probe failed: %v", findErr)
		}
		if window != 0 {
			t.Logf("found Ghost FTP window 0x%x: %s", window, name)
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("mapped Ghost FTP production window was not found")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
