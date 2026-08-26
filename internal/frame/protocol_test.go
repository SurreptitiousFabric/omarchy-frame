package frame

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestDialWSHandshakeAndRoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan error, 1)
	go func() {
		c, e := ln.Accept()
		if e != nil {
			done <- e
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		line, e := r.ReadString('\n')
		if e != nil {
			done <- e
			return
		}
		if !strings.HasPrefix(line, "GET /socket?q=1 HTTP/1.1") {
			done <- io.ErrUnexpectedEOF
			return
		}
		key := ""
		for {
			line, e = r.ReadString('\n')
			if e != nil {
				done <- e
				return
			}
			if line == "\r\n" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
				key = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			}
		}
		h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		_, e = io.WriteString(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "+base64.StdEncoding.EncodeToString(h[:])+"\r\n\r\n")
		if e == nil {
			_, e = c.Write(serverFrame(1, []byte("ready")))
		}
		done <- e
	}()
	w, err := dialWS("ws://"+ln.Addr().String()+"/socket?q=1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	p, err := w.readText(time.Second)
	if err != nil || string(p) != "ready" {
		t.Fatalf("payload=%q err=%v", p, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDialWSRejectsBadHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e == nil {
			defer c.Close()
			r := bufio.NewReader(c)
			for {
				line, x := r.ReadString('\n')
				if x != nil || line == "\r\n" {
					break
				}
			}
			_, _ = io.WriteString(c, "HTTP/1.1 101 Switching Protocols\r\nSec-WebSocket-Accept: wrong\r\n\r\n")
		}
	}()
	if _, err := dialWS("ws://"+ln.Addr().String(), time.Second); err == nil || !strings.Contains(err.Error(), "invalid WebSocket handshake") {
		t.Fatalf("got %v", err)
	}
}

func serverFrame(op byte, payload []byte) []byte {
	h := []byte{0x80 | op}
	if len(payload) < 126 {
		h = append(h, byte(len(payload)))
	} else {
		h = append(h, 126, byte(len(payload)>>8), byte(len(payload)))
	}
	return append(h, payload...)
}

func readClientFrame(t *testing.T, r *bufio.Reader) (byte, []byte) {
	t.Helper()
	a, _ := r.ReadByte()
	b, _ := r.ReadByte()
	n := int(b & 0x7f)
	if n == 126 {
		var x uint16
		_ = binary.Read(r, binary.BigEndian, &x)
		n = int(x)
	}
	if b&0x80 == 0 {
		t.Fatal("client frame was not masked")
	}
	mask := make([]byte, 4)
	_, _ = io.ReadFull(r, mask)
	p := make([]byte, n)
	_, _ = io.ReadFull(r, p)
	for i := range p {
		p[i] ^= mask[i%4]
	}
	return a & 0x0f, p
}

func TestWriteTextProducesMaskedFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	done := make(chan error, 1)
	go func() { done <- w.writeText([]byte("hello")) }()
	op, p := readClientFrame(t, bufio.NewReader(server))
	if op != 1 || string(p) != "hello" {
		t.Fatalf("op=%d payload=%q", op, p)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestReadTextHandlesPingThenText(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		r := bufio.NewReader(server)
		_, _ = server.Write(serverFrame(9, []byte("p")))
		readClientFrame(t, r)
		_, _ = server.Write(serverFrame(1, []byte(` {"ok":true} `)))
	}()
	p, err := w.readText(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != `{"ok":true}` {
		t.Fatalf("%q", p)
	}
}

func TestReadTextRejectsOversizedFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() { _, _ = server.Write([]byte{0x81, 127, 0, 0, 0, 0, 1, 0, 0, 0}) }()
	if _, err := w.readText(time.Second); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("got %v", err)
	}
}

func TestReadTextCloseAndExtendedPayload(t *testing.T) {
	for _, tc := range []struct {
		frame []byte
		want  string
		eof   bool
	}{{serverFrame(8, nil), "", true}, {serverFrame(1, []byte(strings.Repeat("x", 130))), strings.Repeat("x", 130), false}} {
		client, server := net.Pipe()
		w := &wsConn{Conn: client, r: bufio.NewReader(client)}
		go func(b []byte) { _, _ = server.Write(b) }(tc.frame)
		p, err := w.readText(time.Second)
		if tc.eof && err != io.EOF {
			t.Fatalf("want EOF, got %v", err)
		}
		if !tc.eof && (err != nil || string(p) != tc.want) {
			t.Fatalf("len=%d err=%v", len(p), err)
		}
		client.Close()
		server.Close()
	}
}

func TestRemoteURLAndAvahiParser(t *testing.T) {
	u := remoteURL(Config{IP: "192.168.1.8", Token: "a+b"}, "samsung.remote.control")
	if !strings.Contains(u, "token=a%2Bb") {
		t.Fatalf("token not encoded: %s", u)
	}
	line := `=;wlan0;IPv4;Samsung\032The\032Frame;AirPlay Remote Video;local;Samsung.local;192.168.1.88;7000;"manufacturer=Samsung" "model=LS03B"`
	infoFn := func(ip string) (APIInfo, error) {
		var x APIInfo
		x.Device.Name = "Frame"
		x.Device.ModelName = "QE55LS03B"
		x.Device.WifiMac = "00:11:22:33:44:55"
		return x, nil
	}
	got := parseAvahi([]byte(line+"\n"+line), infoFn)
	if len(got) != 1 || got[0].IP != "192.168.1.88" {
		t.Fatalf("%#v", got)
	}
	bad := parseAvahi([]byte(strings.Replace(line, "192.168.1.88", "8.8.8.8", 1)), func(ip string) (APIInfo, error) { t.Fatalf("public address reached verifier"); return APIInfo{}, nil })
	if len(bad) != 0 {
		t.Fatal("accepted public discovery record")
	}
}

func TestRemotePayloadIsJSON(t *testing.T) {
	p := map[string]any{"method": "ms.remote.control", "params": map[string]any{"DataOfCmd": "KEY_HOME"}}
	if _, err := json.Marshal(p); err != nil {
		t.Fatal(err)
	}
}
