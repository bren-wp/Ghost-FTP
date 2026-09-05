//go:build linux

package desktop

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	x11KeyPress        = 2
	x11ButtonPress     = 4
	x11Expose          = 12
	x11DestroyNotify   = 17
	x11ConfigureNotify = 22
	x11ClientMessage   = 33
	x11ExposureMask    = 1 << 15
	x11KeyPressMask    = 1 << 0
	x11ButtonPressMask = 1 << 2
	x11StructureMask   = 1 << 17
	x11ShiftMask       = 1 << 0
	x11ControlMask     = 1 << 2
	x11AtomAtom        = 4
	x11AtomString      = 31
	x11AtomWMName      = 39
	x11AtomWMClass     = 67
	x11InputOutput     = 1
	x11CopyFromParent  = 0
	x11ZPixmap         = 2
	x11KeyBackSpace    = 0xff08
	x11KeyTab          = 0xff09
	x11KeyReturn       = 0xff0d
	x11KeyEscape       = 0xff1b
	x11KeyLeft         = 0xff51
	x11KeyUp           = 0xff52
	x11KeyRight        = 0xff53
	x11KeyDown         = 0xff54
	x11KeyDelete       = 0xffff
)

type x11Display struct {
	number int
	raw    string
}

type x11Visual struct {
	id        uint32
	redMask   uint32
	greenMask uint32
	blueMask  uint32
}

type x11Event struct {
	Type   byte
	Detail byte
	X      int
	Y      int
	State  uint16
	Width  int
	Height int
	Atom   uint32
	Raw    [32]byte
}

type x11Client struct {
	conn      net.Conn
	reader    *bufio.Reader
	mu        sync.Mutex
	order     binary.ByteOrder
	sequence  uint16
	ridBase   uint32
	ridMask   uint32
	ridShift  uint
	ridNext   uint32
	root      uint32
	rootDepth byte
	visual    x11Visual
	window    uint32
	gc        uint32
	font      uint32
	wmDelete  uint32
	width     int
	height    int
	minKey    byte
	maxKey    byte
	keysyms   map[byte][]uint32
	closed    bool
}

func parseLocalDisplay(raw string) (x11Display, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return x11Display{}, errors.New("DISPLAY is not set")
	}
	value := raw
	if strings.HasPrefix(value, "unix:") {
		value = strings.TrimPrefix(value, "unix:")
	}
	if strings.HasPrefix(value, "localhost:") {
		value = ":" + strings.TrimPrefix(value, "localhost:")
	}
	if strings.HasPrefix(value, "127.0.0.1:") {
		value = ":" + strings.TrimPrefix(value, "127.0.0.1:")
	}
	if !strings.HasPrefix(value, ":") {
		return x11Display{}, errors.New("remote X11 displays are blocked by the privacy policy")
	}
	value = strings.TrimPrefix(value, ":")
	displayPart, _, _ := strings.Cut(value, ".")
	number, err := strconv.Atoi(displayPart)
	if err != nil || number < 0 || number > 65535 {
		return x11Display{}, errors.New("DISPLAY has an invalid local display number")
	}
	return x11Display{number: number, raw: raw}, nil
}

func readU16BE(r *bufio.Reader) (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}

func readXAuthorityField(r *bufio.Reader, max int) ([]byte, error) {
	n, err := readU16BE(r)
	if err != nil {
		return nil, err
	}
	if int(n) > max {
		return nil, errors.New("Xauthority field is too large")
	}
	data := make([]byte, int(n))
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func xAuthorityPath() string {
	if configured := strings.TrimSpace(os.Getenv("XAUTHORITY")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".Xauthority")
}

func loadXAuthority(display x11Display) ([]byte, error) {
	filePath := xAuthorityPath()
	if filePath == "" {
		return nil, nil
	}
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Xauthority must be a regular non-symlink file")
	}
	if info.Size() < 0 || info.Size() > 4<<20 {
		return nil, errors.New("Xauthority file size is not accepted")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(io.LimitReader(f, 4<<20))
	wantedNumber := strconv.Itoa(display.number)
	for {
		family, err := readU16BE(r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		address, err := readXAuthorityField(r, 4096)
		if err != nil {
			return nil, err
		}
		number, err := readXAuthorityField(r, 64)
		if err != nil {
			return nil, err
		}
		name, err := readXAuthorityField(r, 128)
		if err != nil {
			return nil, err
		}
		data, err := readXAuthorityField(r, 4096)
		if err != nil {
			return nil, err
		}
		_ = address
		if (family == 256 || family == 65535 || family == 0) && string(number) == wantedNumber && string(name) == "MIT-MAGIC-COOKIE-1" {
			return append([]byte(nil), data...), nil
		}
	}
	return nil, errors.New("no matching MIT-MAGIC-COOKIE-1 entry was found in Xauthority")
}

func pad4(n int) int { return (n + 3) &^ 3 }

func x11Handshake(display x11Display, cookie []byte) []byte {
	name := []byte(nil)
	if len(cookie) != 0 {
		name = []byte("MIT-MAGIC-COOKIE-1")
	}
	buf := make([]byte, 12+pad4(len(name))+pad4(len(cookie)))
	buf[0] = 'l'
	binary.LittleEndian.PutUint16(buf[2:4], 11)
	binary.LittleEndian.PutUint16(buf[4:6], 0)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(len(name)))
	binary.LittleEndian.PutUint16(buf[8:10], uint16(len(cookie)))
	off := 12
	copy(buf[off:off+len(name)], name)
	off += pad4(len(name))
	copy(buf[off:off+len(cookie)], cookie)
	return buf
}

func trailingMaskZeros(mask uint32) uint {
	if mask == 0 {
		return 0
	}
	var shift uint
	for mask&1 == 0 {
		shift++
		mask >>= 1
	}
	return shift
}

func connectLocalX11() (*x11Client, error) {
	display, err := parseLocalDisplay(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, err
	}
	cookie, err := loadXAuthority(display)
	if err != nil {
		// Xvfb and some local sessions explicitly disable access control. Only
		// allow an empty handshake when there is no Xauthority file at all;
		// malformed or mismatched authority state fails closed.
		return nil, err
	}
	socket := fmt.Sprintf("/tmp/.X11-unix/X%d", display.number)
	conn, err := net.DialTimeout("unix", socket, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("local X11 socket is unavailable: %w", err)
	}
	client := &x11Client{conn: conn, reader: bufio.NewReader(conn), order: binary.LittleEndian, keysyms: make(map[byte][]uint32)}
	if err := client.writeRaw(x11Handshake(display, cookie)); err != nil {
		conn.Close()
		return nil, err
	}
	var header [8]byte
	if _, err := io.ReadFull(client.reader, header[:]); err != nil {
		conn.Close()
		return nil, err
	}
	bodyLen := int(binary.LittleEndian.Uint16(header[6:8])) * 4
	if bodyLen < 0 || bodyLen > 16<<20 {
		conn.Close()
		return nil, errors.New("X11 setup response is too large")
	}
	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(client.reader, body); err != nil {
		conn.Close()
		return nil, err
	}
	if header[0] != 1 {
		reasonLen := int(header[1])
		if reasonLen > len(body) {
			reasonLen = len(body)
		}
		reason := strings.TrimSpace(string(body[:reasonLen]))
		if reason == "" {
			reason = "X11 authorization failed"
		}
		conn.Close()
		return nil, errors.New(reason)
	}
	if err := client.parseSetup(body); err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (x *x11Client) parseSetup(body []byte) error {
	if len(body) < 32 {
		return errors.New("X11 setup response is truncated")
	}
	x.ridBase = x.order.Uint32(body[4:8])
	x.ridMask = x.order.Uint32(body[8:12])
	x.ridShift = trailingMaskZeros(x.ridMask)
	x.ridNext = 1
	vendorLen := int(x.order.Uint16(body[16:18]))
	numScreens := int(body[20])
	numFormats := int(body[21])
	x.minKey = body[26]
	x.maxKey = body[27]
	off := 32 + pad4(vendorLen) + numFormats*8
	if numScreens < 1 || off+40 > len(body) {
		return errors.New("X11 setup does not contain a usable screen")
	}
	screen := body[off:]
	x.root = x.order.Uint32(screen[0:4])
	x.visual.id = x.order.Uint32(screen[32:36])
	x.rootDepth = screen[38]
	depthCount := int(screen[39])
	off += 40
	found := false
	for d := 0; d < depthCount; d++ {
		if off+8 > len(body) {
			return errors.New("X11 depth table is truncated")
		}
		visualCount := int(x.order.Uint16(body[off+2 : off+4]))
		off += 8
		for v := 0; v < visualCount; v++ {
			if off+24 > len(body) {
				return errors.New("X11 visual table is truncated")
			}
			visualID := x.order.Uint32(body[off : off+4])
			if visualID == x.visual.id {
				x.visual.redMask = x.order.Uint32(body[off+8 : off+12])
				x.visual.greenMask = x.order.Uint32(body[off+12 : off+16])
				x.visual.blueMask = x.order.Uint32(body[off+16 : off+20])
				found = true
			}
			off += 24
		}
	}
	if !found || x.visual.redMask == 0 || x.visual.greenMask == 0 || x.visual.blueMask == 0 {
		return errors.New("X11 root TrueColor visual is unavailable")
	}
	return nil
}

func (x *x11Client) nextID() (uint32, error) {
	if x.ridMask == 0 {
		return 0, errors.New("X11 resource mask is empty")
	}
	value := (x.ridNext << x.ridShift) & x.ridMask
	if value == 0 {
		x.ridNext++
		value = (x.ridNext << x.ridShift) & x.ridMask
	}
	if value == 0 {
		return 0, errors.New("X11 resource IDs are exhausted")
	}
	x.ridNext++
	return x.ridBase | value, nil
}

func scaleMaskComponent(value byte, mask uint32) uint32 {
	if mask == 0 {
		return 0
	}
	shift := trailingMaskZeros(mask)
	max := mask >> shift
	scaled := (uint32(value)*max + 127) / 255
	return (scaled << shift) & mask
}

func (x *x11Client) pixel(c RGB) uint32 {
	return scaleMaskComponent(c.R, x.visual.redMask) |
		scaleMaskComponent(c.G, x.visual.greenMask) |
		scaleMaskComponent(c.B, x.visual.blueMask)
}

func (x *x11Client) writeRaw(data []byte) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.closed {
		return io.ErrClosedPipe
	}
	for len(data) != 0 {
		n, err := x.conn.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func (x *x11Client) request(data []byte) error {
	if len(data) < 4 || len(data)%4 != 0 {
		return errors.New("invalid X11 request alignment")
	}
	x.order.PutUint16(data[2:4], uint16(len(data)/4))
	x.sequence++
	return x.writeRaw(data)
}

func (x *x11Client) openFont(name string) (uint32, error) {
	id, err := x.nextID()
	if err != nil {
		return 0, err
	}
	nameBytes := []byte(name)
	buf := make([]byte, 12+pad4(len(nameBytes)))
	buf[0] = 45
	x.order.PutUint32(buf[4:8], id)
	x.order.PutUint16(buf[8:10], uint16(len(nameBytes)))
	copy(buf[12:], nameBytes)
	if err := x.request(buf); err != nil {
		return 0, err
	}
	return id, nil
}

func (x *x11Client) internAtom(name string) (uint32, error) {
	nameBytes := []byte(name)
	buf := make([]byte, 8+pad4(len(nameBytes)))
	buf[0] = 16
	x.order.PutUint16(buf[4:6], uint16(len(nameBytes)))
	copy(buf[8:], nameBytes)
	if err := x.request(buf); err != nil {
		return 0, err
	}
	reply, err := x.readReply()
	if err != nil {
		return 0, err
	}
	return x.order.Uint32(reply[8:12]), nil
}

func (x *x11Client) readReply() ([]byte, error) {
	var header [32]byte
	if _, err := io.ReadFull(x.reader, header[:]); err != nil {
		return nil, err
	}
	if header[0] == 0 {
		return nil, fmt.Errorf("X11 protocol error code %d", header[1])
	}
	if header[0] != 1 {
		return nil, errors.New("unexpected X11 event while waiting for a reply")
	}
	extra := int(x.order.Uint32(header[4:8])) * 4
	if extra < 0 || extra > 16<<20 {
		return nil, errors.New("X11 reply is too large")
	}
	result := make([]byte, 32+extra)
	copy(result, header[:])
	if extra != 0 {
		if _, err := io.ReadFull(x.reader, result[32:]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (x *x11Client) queryKeyboardMapping() error {
	if x.maxKey < x.minKey {
		return errors.New("X11 keyboard map range is invalid")
	}
	count := int(x.maxKey-x.minKey) + 1
	if count > 255 {
		return errors.New("X11 keyboard map range is too large")
	}
	buf := make([]byte, 8)
	buf[0] = 101
	buf[1] = x.minKey
	buf[4] = byte(count)
	if err := x.request(buf); err != nil {
		return err
	}
	reply, err := x.readReply()
	if err != nil {
		return err
	}
	per := int(reply[1])
	if per < 1 || per > 16 || len(reply) < 32+count*per*4 {
		return errors.New("X11 keyboard mapping reply is invalid")
	}
	for i := 0; i < count; i++ {
		values := make([]uint32, per)
		for j := 0; j < per; j++ {
			off := 32 + (i*per+j)*4
			values[j] = x.order.Uint32(reply[off : off+4])
		}
		x.keysyms[byte(int(x.minKey)+i)] = values
	}
	return nil
}

func (x *x11Client) createWindow(width, height int, title string) error {
	window, err := x.nextID()
	if err != nil {
		return err
	}
	x.window = window
	x.width, x.height = width, height
	font, err := x.openFont("fixed")
	if err != nil {
		return err
	}
	x.font = font
	gc, err := x.nextID()
	if err != nil {
		return err
	}
	x.gc = gc

	valueMask := uint32((1 << 1) | (1 << 3) | (1 << 11))
	buf := make([]byte, 44)
	buf[0] = 1
	buf[1] = x.rootDepth
	x.order.PutUint32(buf[4:8], window)
	x.order.PutUint32(buf[8:12], x.root)
	x.order.PutUint16(buf[12:14], uint16(40))
	x.order.PutUint16(buf[14:16], uint16(30))
	x.order.PutUint16(buf[16:18], uint16(width))
	x.order.PutUint16(buf[18:20], uint16(height))
	x.order.PutUint16(buf[20:22], 0)
	x.order.PutUint16(buf[22:24], x11InputOutput)
	x.order.PutUint32(buf[24:28], x.visual.id)
	x.order.PutUint32(buf[28:32], valueMask)
	x.order.PutUint32(buf[32:36], x.pixel(premiumTheme.Window))
	x.order.PutUint32(buf[36:40], x.pixel(premiumTheme.Border))
	events := uint32(x11ExposureMask | x11KeyPressMask | x11ButtonPressMask | x11StructureMask)
	x.order.PutUint32(buf[40:44], events)
	if err := x.request(buf); err != nil {
		return err
	}

	gcMask := uint32((1 << 2) | (1 << 3) | (1 << 14))
	gcReq := make([]byte, 24)
	gcReq[0] = 55
	x.order.PutUint32(gcReq[4:8], gc)
	x.order.PutUint32(gcReq[8:12], window)
	x.order.PutUint32(gcReq[12:16], gcMask)
	x.order.PutUint32(gcReq[16:20], x.pixel(premiumTheme.Text))
	x.order.PutUint32(gcReq[20:24], x.pixel(premiumTheme.Window))
	// Font is supplied in a second ChangeGC request because the fixed-size
	// buffer above contains only the two color values.
	if err := x.request(gcReq); err != nil {
		return err
	}
	fontReq := make([]byte, 16)
	fontReq[0] = 56
	x.order.PutUint32(fontReq[4:8], gc)
	x.order.PutUint32(fontReq[8:12], 1<<14)
	x.order.PutUint32(fontReq[12:16], font)
	if err := x.request(fontReq); err != nil {
		return err
	}

	if err := x.changeProperty8(x11AtomWMName, x11AtomString, []byte(title)); err != nil {
		return err
	}
	wmClass := []byte("ghostftp\x00GhostFTP\x00")
	if err := x.changeProperty8(x11AtomWMClass, x11AtomString, wmClass); err != nil {
		return err
	}
	wmProtocols, err := x.internAtom("WM_PROTOCOLS")
	if err != nil {
		return err
	}
	wmDelete, err := x.internAtom("WM_DELETE_WINDOW")
	if err != nil {
		return err
	}
	x.wmDelete = wmDelete
	if err := x.changeProperty32(wmProtocols, x11AtomAtom, []uint32{wmDelete}); err != nil {
		return err
	}
	if err := x.queryKeyboardMapping(); err != nil {
		return err
	}
	mapReq := make([]byte, 8)
	mapReq[0] = 8
	x.order.PutUint32(mapReq[4:8], window)
	return x.request(mapReq)
}

func (x *x11Client) changeProperty8(property, typ uint32, data []byte) error {
	buf := make([]byte, 24+pad4(len(data)))
	buf[0] = 18
	buf[1] = 0
	x.order.PutUint32(buf[4:8], x.window)
	x.order.PutUint32(buf[8:12], property)
	x.order.PutUint32(buf[12:16], typ)
	buf[16] = 8
	x.order.PutUint32(buf[20:24], uint32(len(data)))
	copy(buf[24:], data)
	return x.request(buf)
}

func (x *x11Client) changeProperty32(property, typ uint32, data []uint32) error {
	buf := make([]byte, 24+len(data)*4)
	buf[0] = 18
	buf[1] = 0
	x.order.PutUint32(buf[4:8], x.window)
	x.order.PutUint32(buf[8:12], property)
	x.order.PutUint32(buf[12:16], typ)
	buf[16] = 32
	x.order.PutUint32(buf[20:24], uint32(len(data)))
	for i, value := range data {
		x.order.PutUint32(buf[24+i*4:28+i*4], value)
	}
	return x.request(buf)
}

func (x *x11Client) changeGC(foreground, background RGB) error {
	buf := make([]byte, 20)
	buf[0] = 56
	x.order.PutUint32(buf[4:8], x.gc)
	x.order.PutUint32(buf[8:12], (1<<2)|(1<<3))
	x.order.PutUint32(buf[12:16], x.pixel(foreground))
	x.order.PutUint32(buf[16:20], x.pixel(background))
	return x.request(buf)
}

func (x *x11Client) fillRect(left, top, width, height int, color RGB) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	if err := x.changeGC(color, color); err != nil {
		return err
	}
	buf := make([]byte, 20)
	buf[0] = 70
	x.order.PutUint32(buf[4:8], x.window)
	x.order.PutUint32(buf[8:12], x.gc)
	x.order.PutUint16(buf[12:14], uint16(int16(left)))
	x.order.PutUint16(buf[14:16], uint16(int16(top)))
	x.order.PutUint16(buf[16:18], uint16(width))
	x.order.PutUint16(buf[18:20], uint16(height))
	return x.request(buf)
}

func (x *x11Client) strokeRect(left, top, width, height int, color RGB) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	if err := x.changeGC(color, premiumTheme.Window); err != nil {
		return err
	}
	buf := make([]byte, 20)
	buf[0] = 67
	x.order.PutUint32(buf[4:8], x.window)
	x.order.PutUint32(buf[8:12], x.gc)
	x.order.PutUint16(buf[12:14], uint16(int16(left)))
	x.order.PutUint16(buf[14:16], uint16(int16(top)))
	x.order.PutUint16(buf[16:18], uint16(width))
	x.order.PutUint16(buf[18:20], uint16(height))
	return x.request(buf)
}

func x11TextBytes(text string) []byte {
	var b strings.Builder
	for _, r := range text {
		if r >= 32 && r <= 126 {
			b.WriteRune(r)
		} else if r == '\t' {
			b.WriteByte(' ')
		} else {
			b.WriteByte('?')
		}
		if b.Len() >= 240 {
			break
		}
	}
	return []byte(b.String())
}

func (x *x11Client) text(left, baseline int, text string, foreground, background RGB) error {
	data := x11TextBytes(text)
	if len(data) == 0 {
		return nil
	}
	if err := x.changeGC(foreground, background); err != nil {
		return err
	}
	buf := make([]byte, 16+pad4(len(data)))
	buf[0] = 76
	buf[1] = byte(len(data))
	x.order.PutUint32(buf[4:8], x.window)
	x.order.PutUint32(buf[8:12], x.gc)
	x.order.PutUint16(buf[12:14], uint16(int16(left)))
	x.order.PutUint16(buf[14:16], uint16(int16(baseline)))
	copy(buf[16:], data)
	return x.request(buf)
}

func (x *x11Client) nextEvent() (x11Event, error) {
	var raw [32]byte
	if _, err := io.ReadFull(x.reader, raw[:]); err != nil {
		return x11Event{}, err
	}
	typ := raw[0] & 0x7f
	if typ == 0 {
		return x11Event{}, fmt.Errorf("X11 asynchronous protocol error code %d", raw[1])
	}
	e := x11Event{Type: typ, Detail: raw[1], Raw: raw}
	switch typ {
	case x11KeyPress, x11ButtonPress:
		e.X = int(int16(x.order.Uint16(raw[24:26])))
		e.Y = int(int16(x.order.Uint16(raw[26:28])))
		e.State = x.order.Uint16(raw[28:30])
	case x11ConfigureNotify:
		e.Width = int(x.order.Uint16(raw[20:22]))
		e.Height = int(x.order.Uint16(raw[22:24]))
	case x11ClientMessage:
		e.Atom = x.order.Uint32(raw[12:16])
	}
	return e, nil
}

func (x *x11Client) keysym(keycode byte, state uint16) uint32 {
	values := x.keysyms[keycode]
	if len(values) == 0 {
		return 0
	}
	index := 0
	if state&x11ShiftMask != 0 && len(values) > 1 && values[1] != 0 {
		index = 1
	}
	return values[index]
}

func (x *x11Client) close() {
	x.mu.Lock()
	if x.closed {
		x.mu.Unlock()
		return
	}
	x.closed = true
	conn := x.conn
	x.mu.Unlock()
	_ = conn.Close()
}
