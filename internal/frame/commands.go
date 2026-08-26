package frame

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Run(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, errors.New("usage: frame-controller <discover|configure|status|key|hold|rotate|wake|apps|launch|art>")
	}
	c, e := loadConfig()
	if e != nil {
		return nil, e
	}
	cmd := args[0]
	switch cmd {
	case "discover":
		ds, e := discover()
		return map[string]any{"ok": e == nil, "devices": ds}, e
	case "configure":
		if len(args) < 2 {
			return nil, errors.New("configure requires an IP")
		}
		if net.ParseIP(args[1]) == nil {
			return nil, errors.New("invalid IP")
		}
		info, e := getInfo(args[1])
		if e != nil {
			return nil, e
		}
		c = Config{IP: args[1], MAC: info.Device.WifiMac, Name: info.Device.Name, Model: info.Device.ModelName}
		e = saveConfig(c)
		return map[string]any{"ok": e == nil, "device": c.public()}, e
	}
	if c.IP == "" {
		return nil, errors.New("no TV configured; run discovery first")
	}
	switch cmd {
	case "status":
		info, e := getInfo(c.IP)
		if e != nil {
			return map[string]any{"ok": true, "online": false, "device": c.public()}, nil
		}
		return map[string]any{"ok": true, "online": true, "device": c.public(), "power": info.Device.PowerState, "model": info.Device.ModelName, "capabilities": capabilities()}, nil
	case "wake":
		e = wake(c)
		return map[string]any{"ok": e == nil, "message": "Wake packet sent"}, e
	case "key":
		if len(args) < 2 {
			return nil, errors.New("key requires KEY_NAME")
		}
		e = sendKey(&c, args[1], "click", 0)
		return map[string]any{"ok": e == nil, "message": args[1] + " sent"}, e
	case "hold":
		if len(args) < 2 {
			return nil, errors.New("hold requires KEY_NAME")
		}
		ms := 3000
		if len(args) > 2 {
			ms, _ = strconv.Atoi(args[2])
		}
		if ms < 100 || ms > 10000 {
			return nil, errors.New("hold must be 100-10000 ms")
		}
		e = sendKey(&c, args[1], "hold", time.Duration(ms)*time.Millisecond)
		return map[string]any{"ok": e == nil, "message": args[1] + " held"}, e
	case "rotate":
		e = sendKey(&c, "KEY_MULTI_VIEW", "hold", 3*time.Second)
		return map[string]any{"ok": e == nil, "message": "Rotation toggled"}, e
	case "apps":
		return listApps(c)
	case "launch":
		if len(args) < 2 {
			return nil, errors.New("launch requires app id")
		}
		return launchApp(c, args[1])
	case "art":
		if len(args) < 2 {
			return nil, errors.New("art requires a request type")
		}
		return artRequest(&c, args[1], args[2:])
	default:
		return nil, fmt.Errorf("unknown command %q", cmd)
	}
}

func capabilities() []map[string]string {
	return []map[string]string{
		{"group": "Power", "items": "Wake-on-LAN, power off, Art Mode toggle"}, {"group": "Remote", "items": "navigation, home, back, menu, numbers, color keys, tools, info"}, {"group": "Media", "items": "play, pause, stop, record, rewind, fast-forward, previous, next"}, {"group": "Sound", "items": "volume up/down, mute"}, {"group": "Channels", "items": "channel up/down, guide, channel list"}, {"group": "Sources & apps", "items": "source menu, HDMI keys where firmware accepts them, installed-app listing and launch"}, {"group": "The Frame", "items": "Art API feature detection, current art, art-mode status, available categories, slideshow/rotation requests"}, {"group": "Rotating stand", "items": "three-second Multi View hold toggles portrait/landscape on LS03B"}}
}

func listApps(c Config) (map[string]any, error) {
	client := http.Client{Timeout: 3 * time.Second}
	r, e := client.Get("http://" + c.IP + ":8001/api/v2/applications")
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode/100 != 2 {
		p, err := remoteEvent(&c, "ed.installedApp.get", nil)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				return map[string]any{"ok": true, "supported": false, "apps": []any{}, "message": "Installed-app listing is unavailable on this TV firmware"}, nil
			}
			return nil, err
		}
		if d, ok := p["data"].(map[string]any); ok {
			if apps, ok := d["data"].([]any); ok {
				p["apps"] = apps
			}
		}
		return p, nil
	}
	var apps any
	e = json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&apps)
	return map[string]any{"ok": e == nil, "apps": apps}, e
}
func launchApp(c Config, id string) (map[string]any, error) {
	if strings.ContainsAny(id, "/?#") || id == "" {
		return nil, errors.New("invalid app id")
	}
	req, _ := http.NewRequest("POST", "http://"+c.IP+":8001/api/v2/applications/"+url.PathEscape(id), nil)
	client := http.Client{Timeout: 3 * time.Second}
	r, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode/100 != 2 {
		_, err := remoteEvent(&c, "ed.apps.launch", map[string]any{"appId": id, "action_type": "DEEP_LINK"})
		if err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "message": "App launch requested"}, nil
	}
	return map[string]any{"ok": true, "message": "App launch requested"}, nil
}

func artRequest(c *Config, request string, args []string) (map[string]any, error) {
	w, e := connect(c, "com.samsung.art-app")
	if e != nil {
		return nil, e
	}
	defer w.Close()
	data := map[string]any{"request": request, "id": "omarchy-frame", "request_id": "omarchy-frame"}
	if len(args) > 0 {
		data["value"] = args[0]
	}
	inner, _ := json.Marshal(data)
	outer, _ := json.Marshal(map[string]any{"method": "ms.channel.emit", "params": map[string]any{"event": "art_app_request", "to": "host", "data": string(inner)}})
	if e = w.writeText(outer); e != nil {
		return nil, e
	}
	for i := 0; i < 8; i++ {
		reply, e := w.readText(5 * time.Second)
		if e != nil {
			return nil, e
		}
		var payload map[string]any
		if json.Unmarshal(reply, &payload) != nil {
			continue
		}
		event, _ := payload["event"].(string)
		if event == "ms.channel.ready" || event == "ms.channel.connect" {
			continue
		}
		return map[string]any{"ok": true, "response": payload}, nil
	}
	return nil, errors.New("Art API returned no response")
}

func remoteEvent(c *Config, event string, data map[string]any) (map[string]any, error) {
	w, e := connect(c, "samsung.remote.control")
	if e != nil {
		return nil, e
	}
	defer w.Close()
	params := map[string]any{"event": event, "to": "host"}
	if data != nil {
		params["data"] = data
	}
	b, _ := json.Marshal(map[string]any{"method": "ms.channel.emit", "params": params})
	if e = w.writeText(b); e != nil {
		return nil, e
	}
	for i := 0; i < 8; i++ {
		reply, e := w.readText(4 * time.Second)
		if e != nil {
			return nil, e
		}
		var p map[string]any
		if json.Unmarshal(reply, &p) != nil {
			continue
		}
		got, _ := p["event"].(string)
		if got == "ms.channel.ready" || got == "ms.channel.connect" {
			continue
		}
		if got == event {
			return map[string]any{"ok": true, "event": got, "data": p["data"]}, nil
		}
	}
	return nil, fmt.Errorf("TV returned no %s response", event)
}
