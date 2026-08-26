package frame

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
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
	return serverFragment(true, op, payload)
}

func serverFragment(fin bool, op byte, payload []byte) []byte {
	first := op
	if fin {
		first |= 0x80
	}
	h := []byte{first}
	if len(payload) < 126 {
		h = append(h, byte(len(payload)))
	} else {
		h = append(h, 126, byte(len(payload)>>8), byte(len(payload)))
	}
	return append(h, payload...)
}

type shortWriteConn struct {
	net.Conn
	max int
}

func (c shortWriteConn) Write(p []byte) (int, error) {
	if len(p) > c.max {
		p = p[:c.max]
	}
	return c.Conn.Write(p)
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
	} else if n == 127 {
		var x uint64
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

func TestWriteTextCompletesShortWrites(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: shortWriteConn{Conn: client, max: 2}, r: bufio.NewReader(client)}
	done := make(chan error, 1)
	go func() { done <- w.writeText([]byte("short-write-safe")) }()
	op, p := readClientFrame(t, bufio.NewReader(server))
	if op != 1 || string(p) != "short-write-safe" {
		t.Fatalf("op=%d payload=%q", op, p)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWriteTextLengthEncodingsAndLimits(t *testing.T) {
	for _, size := range []int{130, 70 << 10} {
		client, server := net.Pipe()
		w := &wsConn{Conn: client, r: bufio.NewReader(client)}
		payload := []byte(strings.Repeat("x", size))
		done := make(chan error, 1)
		go func() { done <- w.writeText(payload) }()
		_, got := readClientFrame(t, bufio.NewReader(server))
		if len(got) != size {
			t.Fatalf("size=%d got=%d", size, len(got))
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		_ = client.Close()
		_ = server.Close()
	}
	w := &wsConn{}
	if err := w.writeText(make([]byte, maxWSMessageSize+1)); err == nil {
		t.Fatal("accepted oversized outgoing message")
	}
	if err := w.writeControl(0xA, make([]byte, 126)); err == nil {
		t.Fatal("accepted oversized control payload")
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

func TestReadTextCombinesFragments(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		_, _ = server.Write(serverFragment(false, 1, []byte(` {"ok":`)))
		_, _ = server.Write(serverFragment(true, 0, []byte(`true} `)))
	}()
	p, err := w.readText(time.Second)
	if err != nil || string(p) != `{"ok":true}` {
		t.Fatalf("payload=%q err=%v", p, err)
	}
}

func TestReadTextRejectsMalformedFrames(t *testing.T) {
	for _, frame := range [][]byte{
		serverFragment(true, 0, []byte("orphan")),
		serverFragment(false, 9, []byte("ping")),
		{0x81, 0x81, 0, 0, 0, 0, 'x'},
		serverFragment(true, 2, []byte("binary")),
		serverFragment(true, 3, []byte("unknown")),
		{0xC1, 0},
		append(serverFragment(false, 1, []byte("part")), serverFragment(true, 1, []byte("new"))...),
	} {
		client, server := net.Pipe()
		w := &wsConn{Conn: client, r: bufio.NewReader(client)}
		go func(b []byte) { _, _ = server.Write(b) }(frame)
		if _, err := w.readText(time.Second); err == nil {
			t.Fatalf("accepted malformed frame %x", frame)
		}
		_ = client.Close()
		_ = server.Close()
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

func TestCompletePairingPersistsNewToken(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		message, _ := json.Marshal(map[string]any{"event": "ms.channel.connect", "data": map[string]any{"token": "paired-token"}})
		_, _ = server.Write(serverFrame(1, message))
	}()
	c := Config{IP: "192.168.1.8", Name: "Frame"}
	if err := completePairing(&c, w, time.Second); err != nil {
		t.Fatal(err)
	}
	stored, err := loadConfig()
	if err != nil || stored.Token != "paired-token" || c.Token != "paired-token" {
		t.Fatalf("memory=%#v stored=%#v err=%v", c, stored, err)
	}
}

func TestCompletePairingRejectsUnauthorized(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		message, _ := json.Marshal(map[string]any{"event": "ms.channel.unauthorized"})
		_, _ = server.Write(serverFrame(1, message))
	}()
	if err := completePairing(&Config{}, w, time.Second); err == nil || !strings.Contains(err.Error(), "denied pairing") {
		t.Fatalf("got %v", err)
	}
}

func TestRemotePayloadIsJSON(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	done := make(chan error, 1)
	go func() { done <- sendRemoteKey(w, "KEY_HOME", "Click") }()
	op, payload := readClientFrame(t, bufio.NewReader(server))
	if op != 1 {
		t.Fatalf("unexpected opcode %d", op)
	}
	var message map[string]any
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatal(err)
	}
	params, _ := message["params"].(map[string]any)
	if message["method"] != "ms.remote.control" || params["Cmd"] != "Click" || params["DataOfCmd"] != "KEY_HOME" || params["TypeOfRemote"] != "SendRemoteKey" {
		t.Fatalf("unexpected payload %#v", message)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSendKeyDispatchesOnlyAllowedActions(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	connected := false
	connector := func(_ *Config, channel string) (*wsConn, error) {
		connected = true
		if channel != "samsung.remote.control" {
			t.Fatalf("unexpected channel %q", channel)
		}
		return w, nil
	}
	done := make(chan string, 1)
	go func() {
		_, payload := readClientFrame(t, bufio.NewReader(server))
		var message map[string]any
		_ = json.Unmarshal(payload, &message)
		params, _ := message["params"].(map[string]any)
		done <- fmt.Sprint(params["Cmd"], ":", params["DataOfCmd"])
	}()
	if err := sendKeyWith(&Config{}, "KEY_HOME", "click", 0, connector, time.Sleep); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got != "Click:KEY_HOME" {
		t.Fatalf("got %q", got)
	}
	connected = false
	if err := sendKeyWith(&Config{}, "KEY_FACTORY", "click", 0, connector, time.Sleep); err == nil || connected {
		t.Fatalf("unsafe key: connected=%v err=%v", connected, err)
	}
}
