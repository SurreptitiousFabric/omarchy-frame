package frame

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxThumbnailBatchBytes = int64(64 << 20)
	maxThumbnailCacheBytes = int64(64 << 20)
	maxThumbnailCacheFiles = 200
	maxThumbnailCacheAge   = 30 * 24 * time.Hour
)

type artItem struct {
	ContentID   string `json:"content_id"`
	CategoryID  string `json:"category_id"`
	ContentType string `json:"content_type"`
	ImageDate   string `json:"image_date"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type artRequestFunc func(*Config, string, []string) (map[string]any, error)
type artCommandFunc func(*Config, map[string]any) (map[string]any, error)
type thumbnailFetchFunc func(*Config, []string) (map[string]string, error)

func artGallery(c *Config) (map[string]any, error) {
	return artGalleryWith(c, artRequest, fetchArtThumbnails)
}

func artGalleryWith(c *Config, request artRequestFunc, thumbnails thumbnailFetchFunc) (map[string]any, error) {
	var items []artItem
	var lastErr error
	for _, category := range []string{"MY-C0008", "MY-C0002"} {
		listed, err := request(c, "get_content_list", []string{category})
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
	if current, e := request(c, "get_current_artwork", nil); e == nil {
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
	images, err := thumbnails(c, ids)
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
	return deleteArtWith(c, id, artRequest, artCommand)
}

func deleteArtWith(c *Config, id string, request artRequestFunc, command artCommandFunc) (map[string]any, error) {
	if !safeArtID.MatchString(id) {
		return nil, errors.New("invalid artwork id")
	}
	listed, err := request(c, "get_content_list", []string{"MY-C0002"})
	if err != nil {
		return nil, err
	}
	art, _ := listed["art"].(map[string]any)
	raw, _ := art["content_list"].(string)
	var items []artItem
	if json.Unmarshal([]byte(raw), &items) != nil || !containsArtID(items, id) {
		return nil, errors.New("only photos in My Photos can be deleted")
	}
	if _, err = command(c, map[string]any{"request": "delete_image_list", "content_id_list": []map[string]string{{"content_id": id}}}); err != nil {
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
	if err = sendArtRequest(w, data); err != nil {
		return nil, err
	}
	x, err := waitMatchingArtResponse(w, id, 10, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "art": x}, nil
}

func sendArtRequest(w *wsConn, data map[string]any) error {
	inner, err := json.Marshal(data)
	if err != nil {
		return err
	}
	outer, err := json.Marshal(map[string]any{"method": "ms.channel.emit", "params": map[string]any{"event": "art_app_request", "to": "host", "data": string(inner)}})
	if err != nil {
		return err
	}
	return w.writeText(outer)
}

func waitMatchingArtResponse(w *wsConn, requestID string, attempts int, timeout time.Duration) (map[string]any, error) {
	for i := 0; i < attempts; i++ {
		message, err := w.readText(timeout)
		if err != nil {
			return nil, err
		}
		response, matched, err := matchingArtResponse(message, requestID)
		if err != nil {
			return nil, err
		}
		if matched {
			return response, nil
		}
	}
	return nil, errors.New("Art API returned no matching response")
}

func matchingArtResponse(message []byte, requestID string) (map[string]any, bool, error) {
	var envelope map[string]any
	if json.Unmarshal(message, &envelope) != nil {
		return nil, false, nil
	}
	raw, ok := envelope["data"].(string)
	if !ok {
		return nil, false, nil
	}
	var response map[string]any
	if json.Unmarshal([]byte(raw), &response) != nil {
		return nil, false, nil
	}
	got, _ := response["request_id"].(string)
	if got == "" {
		got, _ = response["id"].(string)
	}
	if got != requestID {
		return nil, false, nil
	}
	if response["event"] == "error" {
		return nil, true, fmt.Errorf("Art request failed: %v", response["error_code"])
	}
	return response, true, nil
}

func fetchArtThumbnails(c *Config, ids []string) (map[string]string, error) {
	return fetchArtThumbnailsWith(c, ids, artCommand)
}

func fetchArtThumbnailsWith(c *Config, ids []string, command artCommandFunc) (map[string]string, error) {
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
	res, err := command(c, map[string]any{"request": "get_thumbnail_list", "content_id_list": req, "conn_info": map[string]any{"d2d_mode": "socket", "connection_id": binary.BigEndian.Uint32(n[:]), "id": d2d}})
	if err != nil {
		return nil, err
	}
	art, _ := res["art"].(map[string]any)
	ci, err := connectionInfo(art)
	if err != nil {
		return nil, err
	}
	secured := boolValue(ci["secured"])
	conn, err := dialTVEndpoint(c, ci, 5*time.Second)
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
	keep := make(map[string]bool, len(images))
	for _, path := range images {
		keep[path] = true
	}
	pruneErr := pruneThumbnailCache(dir, keep, maxThumbnailCacheBytes, maxThumbnailCacheFiles, maxThumbnailCacheAge)
	if err != nil {
		return nil, fmt.Errorf("thumbnail stream (TLS=%t): %w", secured, err)
	}
	if pruneErr != nil {
		return nil, fmt.Errorf("thumbnail cache cleanup: %w", pruneErr)
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
	return receiveThumbnailsWithLimit(r, dir, expected, maxThumbnailBatchBytes)
}

func receiveThumbnailsWithLimit(r io.Reader, dir string, expected map[string]bool, maxBatchBytes int64) (map[string]string, error) {
	out := map[string]string{}
	total := 1
	var batchBytes int64
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
		batchBytes += int64(fileLength)
		if batchBytes > maxBatchBytes {
			return nil, errors.New("thumbnail batch too large")
		}
		total = fileTotal
		b := make([]byte, fileLength)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, fmt.Errorf("thumbnail body: %w", err)
		}
		if !validThumbnail(fileType, b) {
			return nil, errors.New("invalid thumbnail image")
		}
		ext := ".jpg"
		if fileType == "png" {
			ext = ".png"
		}
		p := thumbnailCachePath(dir, fileID, ext)
		if err := os.WriteFile(p, b, 0600); err != nil {
			return nil, err
		}
		out[fileID] = p
	}
	return out, nil
}

func thumbnailCachePath(dir, fileID, ext string) string {
	sum := sha256.Sum256([]byte(fileID))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+ext)
}

type thumbnailCacheFile struct {
	path     string
	size     int64
	modified time.Time
}

func pruneThumbnailCache(dir string, keep map[string]bool, maxBytes int64, maxFiles int, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	now := time.Now()
	var current []thumbnailCacheFile
	var candidates []thumbnailCacheFile
	var firstErr error
	remove := func(path string) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	for _, entry := range entries {
		if !validThumbnailCacheName(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			if firstErr == nil {
				firstErr = infoErr
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		file := thumbnailCacheFile{path: filepath.Join(dir, entry.Name()), size: info.Size(), modified: info.ModTime()}
		if keep[file.path] {
			current = append(current, file)
			continue
		}
		if maxAge > 0 && now.Sub(file.modified) > maxAge {
			remove(file.path)
			continue
		}
		candidates = append(candidates, file)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modified.After(candidates[j].modified) })
	count := len(current)
	var totalBytes int64
	for _, file := range current {
		totalBytes += file.size
	}
	for _, file := range candidates {
		if count+1 > maxFiles || totalBytes+file.size > maxBytes {
			remove(file.path)
			continue
		}
		count++
		totalBytes += file.size
	}
	return firstErr
}

func validThumbnailCacheName(name string) bool {
	ext := filepath.Ext(name)
	if ext != ".jpg" && ext != ".png" {
		return false
	}
	base := strings.TrimSuffix(name, ext)
	if len(base) != 64 {
		return false
	}
	_, err := hex.DecodeString(base)
	return err == nil
}

func validThumbnail(kind string, b []byte) bool {
	if (kind == "jpg" || kind == "jpeg") && len(b) >= 3 {
		return b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff
	}
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	return kind == "png" && len(b) >= len(png) && string(b[:len(png)]) == string(png)
}
