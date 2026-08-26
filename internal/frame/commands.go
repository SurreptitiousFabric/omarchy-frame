package frame

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var safeArtID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var safeArtRequests = map[string]bool{
	"get_artmode_status":   true,
	"get_current_artwork":  true,
	"get_category_list":    true,
	"get_content_list":     true,
	"get_slideshow_status": true,
}

func lanHTTPClient(timeout time.Duration) http.Client {
	return http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func Run(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, errors.New("usage: frame-controller <discover|configure|status|key|hold|rotate|wake|art|gallery|select-art|upload-art|delete-art|slideshow>")
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
		if !isLocalTVIP(net.ParseIP(args[1])) {
			return nil, errors.New("TV address must be a private or link-local IP")
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
	if !isLocalTVIP(net.ParseIP(c.IP)) {
		return nil, errors.New("configured TV address is not a private or link-local IP")
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
	case "art":
		if len(args) < 2 {
			return nil, errors.New("art requires a request type")
		}
		return artRequest(&c, args[1], args[2:])
	case "gallery":
		return artGallery(&c)
	case "select-art":
		if len(args) != 2 {
			return nil, errors.New("select-art requires a content id")
		}
		return selectArt(&c, args[1])
	case "upload-art":
		if len(args) != 2 {
			return nil, errors.New("upload-art requires one local image")
		}
		return uploadArt(&c, args[1])
	case "delete-art":
		if len(args) != 2 {
			return nil, errors.New("delete-art requires one content id")
		}
		return deleteArt(&c, args[1])
	case "slideshow":
		if len(args) != 3 {
			return nil, errors.New("slideshow requires minutes and sequential|shuffle")
		}
		minutes, err := strconv.Atoi(args[1])
		if err != nil || minutes < 0 || minutes > 1440 {
			return nil, errors.New("slideshow minutes must be 0-1440")
		}
		if args[2] != "sequential" && args[2] != "shuffle" {
			return nil, errors.New("slideshow order must be sequential or shuffle")
		}
		return setMyPhotosSlideshow(&c, minutes, args[2] == "shuffle")
	default:
		return nil, fmt.Errorf("unknown command %q", cmd)
	}
}

func capabilities() []map[string]string {
	return []map[string]string{
		{"group": "Power", "items": "Wake-on-LAN, power off, Art Mode toggle"}, {"group": "Remote", "items": "navigation, home, back, menu, numbers, color keys, tools, info"}, {"group": "Media", "items": "play, pause, stop, record, rewind, fast-forward, previous, next"}, {"group": "Sound", "items": "volume up/down, mute"}, {"group": "Channels", "items": "channel up/down, guide, channel list"}, {"group": "Sources", "items": "source menu and HDMI keys where firmware accepts them"}, {"group": "The Frame", "items": "browse artwork; upload, select, and delete My Photos; rotate My Photos as a slideshow"}, {"group": "Rotating stand", "items": "three-second Multi View hold toggles portrait/landscape on LS03B"}}
}

func artRequest(c *Config, request string, args []string) (map[string]any, error) {
	if !safeArtRequests[request] {
		return nil, errors.New("unsupported Art API request")
	}
	if request == "get_content_list" {
		if len(args) != 1 || !safeArtID.MatchString(args[0]) {
			return nil, errors.New("get_content_list requires a valid category id")
		}
	} else if len(args) > 0 {
		return nil, errors.New("unsupported Art API request arguments")
	}
	w, e := connect(c, "com.samsung.art-app")
	if e != nil {
		return nil, e
	}
	defer w.Close()
	id, e := artRequestID()
	if e != nil {
		return nil, e
	}
	data := map[string]any{"request": request, "id": id, "request_id": id}
	if request == "get_content_list" {
		data["category_id"] = args[0]
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
		result := map[string]any{"ok": true, "response": payload}
		if raw, ok := payload["data"].(string); ok {
			var art map[string]any
			if json.Unmarshal([]byte(raw), &art) == nil {
				result["art"] = art
			}
		}
		return result, nil
	}
	return nil, errors.New("Art API returned no response")
}

func artRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// UUID-shaped IDs match Samsung's Art clients; uniqueness is what matters.
	return hex.EncodeToString(b[0:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:16]), nil
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
