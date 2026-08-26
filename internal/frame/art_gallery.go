package frame

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type artItem struct {
	ContentID   string `json:"content_id"`
	CategoryID  string `json:"category_id"`
	ContentType string `json:"content_type"`
	ImageDate   string `json:"image_date"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func artGallery(c *Config) (map[string]any, error) {
	var items []artItem
	var lastErr error
	for _, category := range []string{"MY-C0008", "MY-C0002"} {
		listed, err := artRequest(c, "get_content_list", []string{category})
		if err != nil {
			lastErr = err
			continue
		}
		art, _ := listed["art"].(map[string]any)
		raw, _ := art["content_list"].(string)
		var categoryItems []artItem
		if err := json.Unmarshal([]byte(raw), &categoryItems); err != nil {
			lastErr = errors.New("TV returned an invalid artwork list")
			continue
		}
		items = append(items, categoryItems...)
	}
	if len(items) == 0 && lastErr != nil {
		return nil, lastErr
	}
	if len(items) > 100 {
		return nil, errors.New("TV returned too many artworks")
	}
	items = uniqueArtItems(items)
	currentID := ""
	if current, e := artRequest(c, "get_current_artwork", nil); e == nil {
		if x, ok := current["art"].(map[string]any); ok {
			currentID, _ = x["content_id"].(string)
		}
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if safeArtID.MatchString(item.ContentID) {
			ids = append(ids, item.ContentID)
		}
	}
	images, err := fetchArtThumbnails(c, ids)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !safeArtID.MatchString(item.ContentID) {
			continue
		}
		out = append(out, map[string]any{"id": item.ContentID, "category": item.CategoryID, "type": item.ContentType, "date": item.ImageDate, "width": item.Width, "height": item.Height, "image": images[item.ContentID], "current": item.ContentID == currentID})
	}
	return map[string]any{"ok": true, "items": out, "metadataAvailable": false}, nil
}

func uniqueArtItems(items []artItem) []artItem {
	seen := map[string]bool{}
	out := make([]artItem, 0, len(items))
	for _, item := range items {
		if !safeArtID.MatchString(item.ContentID) || seen[item.ContentID] {
			continue
		}
		seen[item.ContentID] = true
		out = append(out, item)
	}
	return out
}

func selectArt(c *Config, id string) (map[string]any, error) {
	if !safeArtID.MatchString(id) {
		return nil, errors.New("invalid artwork id")
	}
	return artCommand(c, map[string]any{"request": "select_image", "content_id": id, "show": true})
}

func deleteArt(c *Config, id string) (map[string]any, error) {
	if !safeArtID.MatchString(id) {
		return nil, errors.New("invalid artwork id")
	}
	listed, err := artRequest(c, "get_content_list", []string{"MY-C0002"})
	if err != nil {
		return nil, err
	}
	art, _ := listed["art"].(map[string]any)
	raw, _ := art["content_list"].(string)
	var items []artItem
	if json.Unmarshal([]byte(raw), &items) != nil || !containsArtID(items, id) {
		return nil, errors.New("only photos in My Photos can be deleted")
	}
	if _, err = artCommand(c, map[string]any{"request": "delete_image_list", "content_id_list": []map[string]string{{"content_id": id}}}); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "message": "Photo deleted from the TV", "deleted_id": id}, nil
}

func containsArtID(items []artItem, id string) bool {
	for _, item := range items {
		if item.ContentID == id && item.CategoryID == "MY-C0002" {
			return true
		}
	}
	return false
}

func artCommand(c *Config, data map[string]any) (map[string]any, error) {
	id, err := artRequestID()
	if err != nil {
		return nil, err
	}
	data["id"], data["request_id"] = id, id
	w, err := connect(c, "com.samsung.art-app")
	if err != nil {
		return nil, err
	}
	defer w.Close()
	inner, _ := json.Marshal(data)
	outer, _ := json.Marshal(map[string]any{"method": "ms.channel.emit", "params": map[string]any{"event": "art_app_request", "to": "host", "data": string(inner)}})
	if err = w.writeText(outer); err != nil {
		return nil, err
	}
	for i := 0; i < 10; i++ {
		b, e := w.readText(5 * time.Second)
		if e != nil {
			return nil, e
		}
		var p map[string]any
		if json.Unmarshal(b, &p) != nil {
			continue
		}
		raw, ok := p["data"].(string)
		if !ok {
			continue
		}
		var x map[string]any
		if json.Unmarshal([]byte(raw), &x) != nil {
			continue
		}
		got, _ := x["request_id"].(string)
		if got == "" {
			got, _ = x["id"].(string)
		}
		if got != id {
			continue
		}
		if x["event"] == "error" {
			return nil, fmt.Errorf("Art request failed: %v", x["error_code"])
		}
		return map[string]any{"ok": true, "art": x}, nil
	}
	return nil, errors.New("Art API returned no matching response")
}

func fetchArtThumbnails(c *Config, ids []string) (map[string]string, error) {
	req := make([]map[string]string, 0, len(ids))
	for _, id := range ids {
		req = append(req, map[string]string{"content_id": id})
	}
	var n [4]byte
	if _, err := rand.Read(n[:]); err != nil {
		return nil, err
	}
	d2d, err := artRequestID()
	if err != nil {
		return nil, err
	}
	res, err := artCommand(c, map[string]any{"request": "get_thumbnail_list", "content_id_list": req, "conn_info": map[string]any{"d2d_mode": "socket", "connection_id": binary.BigEndian.Uint32(n[:]), "id": d2d}})
	if err != nil {
		return nil, err
	}
	art, _ := res["art"].(map[string]any)
	ci, _ := art["conn_info"].(map[string]any)
	if ci == nil {
		if raw, ok := art["conn_info"].(string); ok {
			_ = json.Unmarshal([]byte(raw), &ci)
		}
	}
	ip, _ := ci["ip"].(string)
	port := portNumber(ci["port"])
	secured := boolValue(ci["secured"])
	endpointIP := net.ParseIP(ip)
	if endpointIP != nil && endpointIP.IsUnspecified() {
		ip = c.IP
		endpointIP = net.ParseIP(ip)
	}
	if endpointIP == nil || !endpointIP.Equal(net.ParseIP(c.IP)) || port < 1 || port > 65535 {
		return nil, errors.New("TV returned an unsafe thumbnail endpoint")
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	var conn net.Conn
	if secured {
		conn, err = tls.DialWithDialer(&d, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)), &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = d.DialContext(context.Background(), "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	}
	if err != nil {
		return nil, fmt.Errorf("thumbnail connection (TLS=%t): %w", secured, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	dir := filepath.Join(filepath.Dir(configPath()), "art-cache")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	for _, id := range ids {
		expected[id] = true
	}
	images, err := receiveThumbnails(bufio.NewReader(conn), dir, expected)
	if err != nil {
		return nil, fmt.Errorf("thumbnail stream (TLS=%t): %w", secured, err)
	}
	return images, nil
}

func portNumber(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func boolValue(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		b, _ := strconv.ParseBool(s)
		return b
	}
	return false
}

func receiveThumbnails(r io.Reader, dir string, expected map[string]bool) (map[string]string, error) {
	out := map[string]string{}
	total := 1
	for i := 0; i < total; i++ {
		var hl uint32
		if err := binary.Read(r, binary.BigEndian, &hl); err != nil {
			if errors.Is(err, io.EOF) && len(out) > 0 {
				return out, nil
			}
			return nil, err
		}
		if hl == 0 || hl > 64<<10 {
			return nil, errors.New("invalid thumbnail header")
		}
		hb := make([]byte, hl)
		if _, err := io.ReadFull(r, hb); err != nil {
			return nil, fmt.Errorf("thumbnail header: %w", err)
		}
		var h map[string]any
		if json.Unmarshal(hb, &h) != nil {
			return nil, errors.New("invalid thumbnail metadata")
		}
		fileID, _ := h["fileID"].(string)
		fileType, _ := h["fileType"].(string)
		fileLength := portNumber(h["fileLength"])
		fileTotal := portNumber(h["total"])
		if !expected[fileID] || fileLength < 1 || fileLength > 8<<20 || fileTotal < 1 || fileTotal > 100 {
			return nil, errors.New("unsafe thumbnail metadata")
		}
		total = fileTotal
		b := make([]byte, fileLength)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, fmt.Errorf("thumbnail body: %w", err)
		}
		if !validThumbnail(fileType, b) {
			return nil, errors.New("invalid thumbnail image")
		}
		sum := sha256.Sum256([]byte(fileID))
		ext := ".jpg"
		if fileType == "png" {
			ext = ".png"
		}
		p := filepath.Join(dir, hex.EncodeToString(sum[:])+ext)
		if err := os.WriteFile(p, b, 0600); err != nil {
			return nil, err
		}
		out[fileID] = p
	}
	return out, nil
}

func validThumbnail(kind string, b []byte) bool {
	if (kind == "jpg" || kind == "jpeg") && len(b) >= 3 {
		return b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff
	}
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	return kind == "png" && len(b) >= len(png) && string(b[:len(png)]) == string(png)
}
