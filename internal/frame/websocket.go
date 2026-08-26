package frame

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

type wsConn struct {
	net.Conn
	r *bufio.Reader
}

const maxWSMessageSize = 8 << 20

func dialWS(raw string, timeout time.Duration) (*wsConn, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := u.Host
	d := net.Dialer{Timeout: timeout}
	var c net.Conn
	if u.Scheme == "wss" {
		c, err = tls.DialWithDialer(&d, "tcp", host, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // Samsung uses a self-signed LAN certificate.
	} else {
		c, err = d.Dial("tcp", host)
	}
	if err != nil {
		return nil, err
	}
	keyBytes := make([]byte, 16)
	if _, err = rand.Read(keyBytes); err != nil {
		c.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\nOrigin: http://localhost\r\n\r\n", path, host, key)
	_ = c.SetDeadline(time.Now().Add(timeout))
	if _, err = io.WriteString(c, req); err != nil {
		c.Close()
		return nil, err
	}
	r := bufio.NewReader(c)
	status, err := r.ReadString('\n')
	if err != nil {
		c.Close()
		return nil, err
	}
	if !strings.Contains(status, " 101 ") {
		c.Close()
		return nil, fmt.Errorf("TV rejected WebSocket: %s", strings.TrimSpace(status))
	}
	accept := ""
	for {
		line, e := r.ReadString('\n')
		if e != nil {
			c.Close()
			return nil, e
		}
		if line == "\r\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			accept = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if accept != base64.StdEncoding.EncodeToString(h[:]) {
		c.Close()
		return nil, errors.New("invalid WebSocket handshake")
	}
	_ = c.SetDeadline(time.Time{})
	return &wsConn{Conn: c, r: r}, nil
}

func (w *wsConn) writeText(payload []byte) error {
	if len(payload) > maxWSMessageSize {
		return errors.New("WebSocket message too large")
	}
	return w.writeMaskedFrame(0x1, payload)
}

func (w *wsConn) writeMaskedFrame(op byte, payload []byte) error {
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	h := []byte{0x80 | op}
	n := len(payload)
	switch {
	case n < 126:
		h = append(h, byte(n)|0x80)
	case n <= 65535:
		h = append(h, 126|0x80, byte(n>>8), byte(n))
	default:
		h = append(h, 127|0x80)
		x := make([]byte, 8)
		binary.BigEndian.PutUint64(x, uint64(n))
		h = append(h, x...)
	}
	h = append(h, mask...)
	data := make([]byte, n)
	for i := range payload {
		data[i] = payload[i] ^ mask[i%4]
	}
	return writeAll(w, append(h, data...))
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(p) {
			return io.ErrUnexpectedEOF
		}
		p = p[n:]
	}
	return nil
}

func (w *wsConn) readText(timeout time.Duration) ([]byte, error) {
	_ = w.SetReadDeadline(time.Now().Add(timeout))
	var message []byte
	fragmented := false
	for {
		fin, op, p, err := w.readFrame()
		if err != nil {
			return nil, err
		}
		switch op {
		case 0x0:
			if !fragmented {
				return nil, errors.New("unexpected WebSocket continuation frame")
			}
			if len(message)+len(p) > maxWSMessageSize {
				return nil, errors.New("WebSocket message too large")
			}
			message = append(message, p...)
			if fin {
				return bytes.TrimSpace(message), nil
			}
		case 0x1:
			if fragmented {
				return nil, errors.New("new WebSocket message before final continuation")
			}
			if fin {
				return bytes.TrimSpace(p), nil
			}
			message = append(message[:0], p...)
			fragmented = true
		case 0x2:
			return nil, errors.New("unsupported WebSocket binary message")
		case 0x8:
			return nil, io.EOF
		case 0x9:
			if err := w.writeControl(0xA, p); err != nil {
				return nil, err
			}
		case 0xA:
			continue
		default:
			return nil, errors.New("unsupported WebSocket frame")
		}
	}
}

func (w *wsConn) readFrame() (bool, byte, []byte, error) {
	a, err := w.r.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}
	b, err := w.r.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}
	if a&0x70 != 0 {
		return false, 0, nil, errors.New("unsupported WebSocket extension bits")
	}
	if b&0x80 != 0 {
		return false, 0, nil, errors.New("masked WebSocket server frame")
	}
	fin := a&0x80 != 0
	op := a & 0x0f
	n := uint64(b & 0x7f)
	if n == 126 {
		var x uint16
		if err = binary.Read(w.r, binary.BigEndian, &x); err != nil {
			return false, 0, nil, err
		}
		n = uint64(x)
	} else if n == 127 {
		if err = binary.Read(w.r, binary.BigEndian, &n); err != nil {
			return false, 0, nil, err
		}
	}
	if n > maxWSMessageSize {
		return false, 0, nil, errors.New("WebSocket frame too large")
	}
	if op >= 0x8 && (!fin || n > 125) {
		return false, 0, nil, errors.New("invalid WebSocket control frame")
	}
	p := make([]byte, n)
	if _, err = io.ReadFull(w.r, p); err != nil {
		return false, 0, nil, err
	}
	return fin, op, p, nil
}

func (w *wsConn) writeControl(op byte, p []byte) error {
	if len(p) > 125 {
		return errors.New("WebSocket control payload too large")
	}
	return w.writeMaskedFrame(op, p)
}
