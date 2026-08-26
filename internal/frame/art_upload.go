package frame

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxUploadSize = 20 << 20

type endpointDialFunc func(*Config, map[string]any, time.Duration) (net.Conn, error)
type artSelectFunc func(*Config, string) (map[string]any, error)

func uploadArt(c *Config, input string) (map[string]any, error) {
	return uploadArtWith(c, input, connect, dialTVEndpoint, selectArt)
}

func uploadArtWith(c *Config, input string, connectArt wsConnectFunc, dialEndpoint endpointDialFunc, selectImage artSelectFunc) (map[string]any, error) {
	f, size, kind, err := openUploadImage(input)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	id, err := artRequestID()
	if err != nil {
		return nil, err
	}
	var randomConnection [4]byte
	if _, err = rand.Read(randomConnection[:]); err != nil {
		return nil, err
	}
	w, err := connectArt(c, "com.samsung.art-app")
	if err != nil {
		return nil, err
	}
	defer w.Close()
	request := map[string]any{
		"request": "send_image", "id": id, "request_id": id,
		"file_type": kind, "file_size": size,
		"image_date": time.Now().Format("2006:01:02 15:04:05"),
		"matte_id":   "none", "portrait_matte_id": "none",
		"conn_info": map[string]any{"d2d_mode": "socket", "connection_id": binary.BigEndian.Uint32(randomConnection[:]), "id": id},
	}
	inner, _ := json.Marshal(request)
	outer, _ := json.Marshal(map[string]any{"method": "ms.channel.emit", "params": map[string]any{"event": "art_app_request", "to": "host", "data": string(inner)}})
	if err = w.writeText(outer); err != nil {
		return nil, err
	}
	ready, err := waitArtEvent(w, id, "ready_to_use", 10*time.Second)
	if err != nil {
		return nil, err
	}
	ci, err := connectionInfo(ready)
	if err != nil {
		return nil, err
	}
	key, _ := ci["key"].(string)
	if key == "" || len(key) > 4096 {
		return nil, errors.New("TV returned invalid upload authorization")
	}
	conn, err := dialEndpoint(c, ci, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("image upload connection: %w", err)
	}
	header, _ := json.Marshal(map[string]any{"num": 0, "total": 1, "fileLength": size, "fileName": "upload", "fileType": kind, "secKey": key, "version": "0.0.1"})
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err = binary.Write(conn, binary.BigEndian, uint32(len(header))); err == nil {
		err = writeAll(conn, header)
	}
	if err == nil {
		_, err = io.CopyN(conn, f, size)
	}
	_ = conn.Close()
	if err != nil {
		return nil, errors.New("image transfer did not complete")
	}
	added, err := waitArtEvent(w, "", "image_added", 20*time.Second)
	if err != nil {
		return nil, err
	}
	contentID, _ := added["content_id"].(string)
	if !safeArtID.MatchString(contentID) {
		return nil, errors.New("TV did not return a valid uploaded artwork id")
	}
	_, selectErr := selectImage(c, contentID)
	return uploadedPhotoResult(contentID, selectErr), nil
}

func openUploadImage(input string) (*os.File, int64, string, error) {
	p, err := localImagePath(input)
	if err != nil {
		return nil, 0, "", err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, "", errors.New("could not open the selected image")
	}
	fail := func(message string) (*os.File, int64, string, error) {
		_ = f.Close()
		return nil, 0, "", errors.New(message)
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 8 || info.Size() > maxUploadSize {
		return fail("image must be a regular JPEG or PNG no larger than 20 MB")
	}
	probe := make([]byte, 8)
	if _, err = io.ReadFull(f, probe); err != nil {
		return fail("could not read the selected image")
	}
	kind := ""
	if validThumbnail("jpg", probe) {
		kind = "jpg"
	} else if validThumbnail("png", probe) {
		kind = "png"
	} else {
		return fail("selected file is not a valid JPEG or PNG")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail("could not read the selected image")
	}
	return f, info.Size(), kind, nil
}

func uploadedPhotoResult(contentID string, selectErr error) map[string]any {
	if selectErr != nil {
		return map[string]any{"ok": true, "message": "Photo uploaded; select it from My Photos", "content_id": contentID, "selected": false}
	}
	return map[string]any{"ok": true, "message": "Photo uploaded and selected", "content_id": contentID, "selected": true}
}

func localImagePath(input string) (string, error) {
	if strings.HasPrefix(input, "file:") {
		u, err := url.Parse(input)
		if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") {
			return "", errors.New("invalid local image URL")
		}
		input = u.Path
	}
	if !filepath.IsAbs(input) || strings.ContainsRune(input, 0) {
		return "", errors.New("image path must be absolute")
	}
	return filepath.Clean(input), nil
}

func setMyPhotosSlideshow(c *Config, minutes int, shuffle bool) (map[string]any, error) {
	value := "off"
	if minutes > 0 {
		value = formatSlideshowMinutes(minutes)
	}
	typ := "slideshow"
	if shuffle {
		typ = "shuffleslideshow"
	}
	return artCommand(c, map[string]any{"request": "set_slideshow_status", "value": value, "category_id": "MY-C0002", "type": typ})
}

func formatSlideshowMinutes(minutes int) string { return strconv.Itoa(minutes) }

func waitArtEvent(w *wsConn, requestID, event string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b, err := w.readText(time.Until(deadline))
		if err != nil {
			return nil, err
		}
		var envelope map[string]any
		if json.Unmarshal(b, &envelope) != nil {
			continue
		}
		raw, ok := envelope["data"].(string)
		if !ok {
			continue
		}
		var x map[string]any
		if json.Unmarshal([]byte(raw), &x) != nil {
			continue
		}
		if x["event"] == "error" {
			return nil, fmt.Errorf("Art request failed: %v", x["error_code"])
		}
		got, _ := x["request_id"].(string)
		if got == "" {
			got, _ = x["id"].(string)
		}
		if x["event"] == event && (requestID == "" || got == requestID) {
			return x, nil
		}
	}
	return nil, fmt.Errorf("Art API returned no %s response", event)
}

func connectionInfo(art map[string]any) (map[string]any, error) {
	ci, _ := art["conn_info"].(map[string]any)
	if ci == nil {
		if raw, ok := art["conn_info"].(string); ok {
			_ = json.Unmarshal([]byte(raw), &ci)
		}
	}
	if ci == nil {
		return nil, errors.New("TV returned no transfer endpoint")
	}
	return ci, nil
}

func dialTVEndpoint(c *Config, ci map[string]any, timeout time.Duration) (net.Conn, error) {
	ip, _ := ci["ip"].(string)
	port := portNumber(ci["port"])
	secured := boolValue(ci["secured"])
	endpointIP := net.ParseIP(ip)
	if endpointIP != nil && endpointIP.IsUnspecified() {
		ip, endpointIP = c.IP, net.ParseIP(c.IP)
	}
	if endpointIP == nil || !endpointIP.Equal(net.ParseIP(c.IP)) || port < 1 || port > 65535 {
		return nil, errors.New("TV returned an unsafe transfer endpoint")
	}
	dialer := net.Dialer{Timeout: timeout}
	if secured {
		return tls.DialWithDialer(&dialer, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)), &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	}
	return dialer.DialContext(context.Background(), "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
}
