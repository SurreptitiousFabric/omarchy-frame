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

func dialWS(raw string, timeout time.Duration) (*wsConn, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := u.Host
	d := net.Dialer{Timeout: timeout}
	var c net.Conn
	if u.Scheme == "wss" {
		c, err = tls.DialWithDialer(&d, "tcp", host, &tls.Config{InsecureSkipVerify: true}) // Samsung uses a self-signed LAN certificate.
	} else {
		c, err = d.Dial("tcp", host)
	}
	if err != nil {
		return nil, err
	}
	keyBytes := make([]byte, 16)
	_, _ = rand.Read(keyBytes)
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
	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	h := []byte{0x81}
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
	_, err := w.Write(append(h, data...))
	return err
}

func (w *wsConn) readText(timeout time.Duration) ([]byte, error) {
	_ = w.SetReadDeadline(time.Now().Add(timeout))
	for {
		a, err := w.r.ReadByte()
		if err != nil {
			return nil, err
		}
		b, err := w.r.ReadByte()
		if err != nil {
			return nil, err
		}
		op := a & 0x0f
		n := uint64(b & 0x7f)
		if n == 126 {
			var x uint16
			if err = binary.Read(w.r, binary.BigEndian, &x); err != nil {
				return nil, err
			}
			n = uint64(x)
		} else if n == 127 {
			if err = binary.Read(w.r, binary.BigEndian, &n); err != nil {
				return nil, err
			}
		}
		var mask []byte
		if b&0x80 != 0 {
			mask = make([]byte, 4)
			if _, err = io.ReadFull(w.r, mask); err != nil {
				return nil, err
			}
		}
		if n > 8<<20 {
			return nil, errors.New("WebSocket frame too large")
		}
		p := make([]byte, n)
		if _, err = io.ReadFull(w.r, p); err != nil {
			return nil, err
		}
		for i := range p {
			if len(mask) > 0 {
				p[i] ^= mask[i%4]
			}
		}
		if op == 0x8 {
			return nil, io.EOF
		}
		if op == 0x9 {
			_ = w.writeControl(0xA, p)
			continue
		}
		if op == 0x1 {
			return bytes.TrimSpace(p), nil
		}
	}
}

func (w *wsConn) writeControl(op byte, p []byte) error {
	if len(p) > 125 {
		p = p[:125]
	}
	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	h := []byte{0x80 | op, 0x80 | byte(len(p))}
	h = append(h, mask...)
	for i := range p {
		p[i] ^= mask[i%4]
	}
	_, e := w.Write(append(h, p...))
	return e
}
