package frame

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

const avahiBrowsePath = "/usr/bin/avahi-browse"

var safeClickKeys = map[string]struct{}{
	"KEY_CAPTION":   {},
	"KEY_CHDOWN":    {},
	"KEY_CHUP":      {},
	"KEY_CH_LIST":   {},
	"KEY_DOWN":      {},
	"KEY_ENTER":     {},
	"KEY_EXIT":      {},
	"KEY_FF":        {},
	"KEY_GUIDE":     {},
	"KEY_HOME":      {},
	"KEY_INFO":      {},
	"KEY_LEFT":      {},
	"KEY_MENU":      {},
	"KEY_MUTE":      {},
	"KEY_PLAYPAUSE": {},
	"KEY_POWER":     {},
	"KEY_PRECH":     {},
	"KEY_PREVIOUS":  {},
	"KEY_REC":       {},
	"KEY_RETURN":    {},
	"KEY_REWIND":    {},
	"KEY_RIGHT":     {},
	"KEY_SOURCE":    {},
	"KEY_STOP":      {},
	"KEY_TOOLS":     {},
	"KEY_UP":        {},
	"KEY_VOLDOWN":   {},
	"KEY_VOLUP":     {},
}

var safeHoldKeys = map[string]struct{}{
	"KEY_MULTI_VIEW": {},
	"KEY_POWER":      {},
}

type wsConnectFunc func(*Config, string) (*wsConn, error)

func safeRemoteKey(key, command string) bool {
	var allowed map[string]struct{}
	switch command {
	case "click":
		allowed = safeClickKeys
	case "hold":
		allowed = safeHoldKeys
	default:
		return false
	}
	_, ok := allowed[key]
	return ok
}

type APIInfo struct {
	Device  struct{ Name, ModelName, IP, WifiMac, PowerState string } `json:"device"`
	Name    string                                                    `json:"name"`
	Version string                                                    `json:"version"`
}

func getInfo(ip string) (APIInfo, error) {
	return getInfoWithClient(ip, "http://"+net.JoinHostPort(ip, "8001")+"/api/v2/", lanHTTPClient(1800*time.Millisecond))
}

func getInfoWithClient(ip, endpoint string, client http.Client) (APIInfo, error) {
	var out APIInfo
	if !isLocalTVIP(net.ParseIP(ip)) {
		return out, errors.New("TV address is not local")
	}
	r, e := client.Get(endpoint)
	if e != nil {
		return out, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return out, fmt.Errorf("TV HTTP %s", r.Status)
	}
	body, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if e != nil {
		return out, e
	}
	if len(body) > 1<<20 {
		return out, errors.New("TV information response too large")
	}
	e = json.Unmarshal(body, &out)
	return out, e
}

func remoteURL(c Config, channel string) string {
	n := base64.StdEncoding.EncodeToString([]byte("Omarchy Frame"))
	q := "name=" + url.QueryEscape(n)
	if c.Token != "" {
		q += "&token=" + url.QueryEscape(c.Token)
	}
	return "wss://" + net.JoinHostPort(c.IP, "8002") + "/api/v2/channels/" + channel + "?" + q
}

func connect(c *Config, channel string) (*wsConn, error) {
	return connectWithHandshakeTimeout(c, channel, 30*time.Second)
}

func connectWithHandshakeTimeout(c *Config, channel string, handshakeTimeout time.Duration) (*wsConn, error) {
	w, e := dialWS(remoteURL(*c, channel), 8*time.Second)
	if e != nil {
		return nil, e
	}
	if e = completePairing(c, w, handshakeTimeout); e != nil {
		w.Close()
		return nil, e
	}
	return w, nil
}

func completePairing(c *Config, w *wsConn, handshakeTimeout time.Duration) error {
	msg, e := w.readText(handshakeTimeout)
	if e != nil {
		return fmt.Errorf("approve Omarchy Frame on the TV: %w", e)
	}
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(msg, &event)
	if event.Event == "ms.channel.unauthorized" {
		return errors.New("TV denied pairing; remove the old authorization in TV settings and retry")
	}
	if event.Data.Token != "" && event.Data.Token != c.Token {
		c.Token = event.Data.Token
		if e = saveConfig(*c); e != nil {
			return e
		}
	}
	return nil
}

func sendKey(c *Config, key, cmd string, hold time.Duration) error {
	return sendKeyWith(c, key, cmd, hold, connect, time.Sleep)
}

func sendKeyWith(c *Config, key, cmd string, hold time.Duration, connectRemote wsConnectFunc, wait func(time.Duration)) error {
	if !safeRemoteKey(key, cmd) {
		return errors.New("unsupported Samsung remote key")
	}
	w, e := connectRemote(c, "samsung.remote.control")
	if e != nil {
		return e
	}
	defer w.Close()
	send := func(action string) error { return sendRemoteKey(w, key, action) }
	if cmd == "hold" {
		return sendHeldKey(send, hold, wait)
	}
	return send("Click")
}

func sendRemoteKey(w *wsConn, key, action string) error {
	p := map[string]any{"method": "ms.remote.control", "params": map[string]any{"Cmd": action, "DataOfCmd": key, "Option": "false", "TypeOfRemote": "SendRemoteKey"}}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return w.writeText(b)
}

func sendHeldKey(send func(string) error, hold time.Duration, wait func(time.Duration)) (err error) {
	if err = send("Press"); err != nil {
		return err
	}
	released := false
	defer func() {
		if released {
			return
		}
		if retryErr := send("Release"); retryErr == nil {
			err = nil
		} else if err == nil {
			err = retryErr
		}
	}()
	wait(hold)
	err = send("Release")
	released = err == nil
	return err
}

func wake(c Config) error {
	p, e := wakePacket(c.MAC)
	if e != nil {
		return e
	}
	conn, e := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4bcast, Port: 9})
	if e != nil {
		return e
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, e = conn.Write(p)
	return e
}

func wakePacket(mac string) ([]byte, error) {
	hw, e := net.ParseMAC(mac)
	if e != nil {
		return nil, errors.New("TV MAC address is not known; discover while the TV is on first")
	}
	if len(hw) != 6 {
		return nil, errors.New("unsupported MAC address")
	}
	p := bytes.Repeat([]byte{0xff}, 6)
	for i := 0; i < 16; i++ {
		p = append(p, hw...)
	}
	return p, nil
}

func discover() ([]Config, error) {
	conn, e := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if e != nil {
		return nil, e
	}
	defer conn.Close()
	msg := "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: urn:samsung.com:device:RemoteControlReceiver:1\r\n\r\n"
	_, e = conn.WriteToUDP([]byte(msg), &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900})
	if e != nil {
		return nil, e
	}
	_ = conn.SetReadDeadline(time.Now().Add(2500 * time.Millisecond))
	seen := map[string]bool{}
	var out []Config
	buf := make([]byte, 4096)
	for {
		_, a, e := conn.ReadFromUDP(buf)
		if e != nil {
			break
		}
		ip := a.IP.String()
		if seen[ip] {
			continue
		}
		seen[ip] = true
		if info, e := getInfo(ip); e == nil && strings.Contains(strings.ToLower(info.Device.ModelName+" "+info.Name), "frame") {
			out = append(out, Config{IP: ip, MAC: info.Device.WifiMac, Name: info.Device.Name, Model: info.Device.ModelName})
		}
	}
	if len(out) == 0 {
		out = discoverAvahi()
	}
	return out, nil
}

// Current Frame firmware commonly advertises AirPlay over mDNS while ignoring
// Samsung's legacy SSDP target. Avahi ships with Omarchy, so use its stable
// parsable output as a fallback without adding a runtime library.
func discoverAvahi() []Config {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, avahiBrowsePath, "-rtp", "_airplay._tcp").Output()
	if err != nil && len(b) == 0 {
		return nil
	}
	return parseAvahi(b, getInfo)
}

func parseAvahi(b []byte, infoFn func(string) (APIInfo, error)) []Config {
	seen := map[string]bool{}
	var out []Config
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Split(line, ";")
		if len(fields) < 10 || fields[0] != "=" || fields[2] != "IPv4" {
			continue
		}
		txt := strings.ToLower(strings.Join(fields[9:], ";"))
		if !strings.Contains(txt, "manufacturer=samsung") && !strings.Contains(strings.ToLower(fields[3]), "samsung") {
			continue
		}
		ip := fields[7]
		if seen[ip] || !isLocalTVIP(net.ParseIP(ip)) {
			continue
		}
		seen[ip] = true
		if info, err := infoFn(ip); err == nil {
			out = append(out, Config{IP: ip, MAC: info.Device.WifiMac, Name: info.Device.Name, Model: info.Device.ModelName})
		}
	}
	return out
}
