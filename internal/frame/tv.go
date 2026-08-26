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
	"regexp"
	"strings"
	"time"
)

var safeKey = regexp.MustCompile(`^KEY_[A-Z0-9_]+$`)

type APIInfo struct {
	Device  struct{ Name, ModelName, IP, WifiMac, PowerState string } `json:"device"`
	Name    string                                                    `json:"name"`
	Version string                                                    `json:"version"`
}

func getInfo(ip string) (APIInfo, error) {
	var out APIInfo
	c := http.Client{Timeout: 1800 * time.Millisecond}
	r, e := c.Get("http://" + ip + ":8001/api/v2/")
	if e != nil {
		return out, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return out, fmt.Errorf("TV HTTP %s", r.Status)
	}
	e = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&out)
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
	w, e := dialWS(remoteURL(*c, channel), 8*time.Second)
	if e != nil {
		return nil, e
	}
	msg, e := w.readText(30 * time.Second)
	if e != nil {
		w.Close()
		return nil, fmt.Errorf("approve Omarchy Frame on the TV: %w", e)
	}
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(msg, &event)
	if event.Event == "ms.channel.unauthorized" {
		w.Close()
		return nil, errors.New("TV denied pairing; remove the old authorization in TV settings and retry")
	}
	if event.Data.Token != "" && event.Data.Token != c.Token {
		c.Token = event.Data.Token
		if e = saveConfig(*c); e != nil {
			w.Close()
			return nil, e
		}
	}
	return w, nil
}

func sendKey(c *Config, key, cmd string, hold time.Duration) error {
	if !safeKey.MatchString(key) {
		return errors.New("invalid Samsung key")
	}
	w, e := connect(c, "samsung.remote.control")
	if e != nil {
		return e
	}
	defer w.Close()
	send := func(action string) error {
		p := map[string]any{"method": "ms.remote.control", "params": map[string]any{"Cmd": action, "DataOfCmd": key, "Option": "false", "TypeOfRemote": "SendRemoteKey"}}
		b, _ := json.Marshal(p)
		return w.writeText(b)
	}
	if cmd == "hold" {
		if e = send("Press"); e != nil {
			return e
		}
		time.Sleep(hold)
		return send("Release")
	}
	return send("Click")
}

func wake(c Config) error {
	hw, e := net.ParseMAC(c.MAC)
	if e != nil {
		return errors.New("TV MAC address is not known; discover while the TV is on first")
	}
	if len(hw) != 6 {
		return errors.New("unsupported MAC address")
	}
	p := bytes.Repeat([]byte{0xff}, 6)
	for i := 0; i < 16; i++ {
		p = append(p, hw...)
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
	b, err := exec.CommandContext(ctx, "avahi-browse", "-rtp", "_airplay._tcp").Output()
	if err != nil && len(b) == 0 {
		return nil
	}
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
		if seen[ip] || net.ParseIP(ip) == nil {
			continue
		}
		seen[ip] = true
		if info, err := getInfo(ip); err == nil {
			out = append(out, Config{IP: ip, MAC: info.Device.WifiMac, Name: info.Device.Name, Model: info.Device.ModelName})
		}
	}
	return out
}
