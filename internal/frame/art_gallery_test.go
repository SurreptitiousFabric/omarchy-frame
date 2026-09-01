package frame

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func thumbnailFrame(t *testing.T, id string, data []byte, num, total int) []byte {
	t.Helper()
	h, err := json.Marshal(map[string]any{"fileLength": len(data), "fileID": id, "fileType": "jpg", "num": num, "total": total})
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(len(h)))
	b.Write(h)
	b.Write(data)
	return b.Bytes()
}

func TestReceiveThumbnails(t *testing.T) {
	b := append(thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2), thumbnailFrame(t, "B", []byte{0xff, 0xd8, 0xff, 2}, 1, 2)...)
	dir := t.TempDir()
	got, err := receiveThumbnails(bytes.NewReader(b), dir, map[string]bool{"A": true, "B": true})
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	for _, p := range got {
		info, e := os.Stat(p)
		if e != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("mode/error for %s: %v", p, e)
		}
	}
}

func TestReceiveThumbnailsRejectsTruncatedBatch(t *testing.T) {
	b := thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2)
	if _, err := receiveThumbnails(bytes.NewReader(b), t.TempDir(), map[string]bool{"A": true, "B": true}); err == nil || !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("got %v", err)
	}
}

func TestReceiveThumbnailsRejectsInconsistentBatchMetadata(t *testing.T) {
	tests := map[string][]byte{
		"out of order":  append(thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2), thumbnailFrame(t, "B", []byte{0xff, 0xd8, 0xff, 2}, 0, 2)...),
		"changed total": append(thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2), thumbnailFrame(t, "B", []byte{0xff, 0xd8, 0xff, 2}, 1, 1)...),
	}
	for name, stream := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := receiveThumbnails(bytes.NewReader(stream), t.TempDir(), map[string]bool{"A": true, "B": true}); err == nil || !strings.Contains(err.Error(), "unsafe thumbnail metadata") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestValidThumbnail(t *testing.T) {
	if !validThumbnail("jpg", []byte{0xff, 0xd8, 0xff}) || !validThumbnail("png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) || validThumbnail("jpg", []byte("no")) {
		t.Fatal("thumbnail signature validation failed")
	}
}

func TestUniqueArtItems(t *testing.T) {
	got := uniqueArtItems([]artItem{{ContentID: "A"}, {ContentID: "A"}, {ContentID: "../bad"}, {ContentID: "B"}})
	if len(got) != 2 || got[0].ContentID != "A" || got[1].ContentID != "B" {
		t.Fatalf("got %#v", got)
	}
}

func TestContainsArtIDOnlyAllowsMyPhotos(t *testing.T) {
	items := []artItem{{ContentID: "mine", CategoryID: "MY-C0002"}, {ContentID: "store", CategoryID: "MY-C0008"}}
	if !containsArtID(items, "mine") || containsArtID(items, "store") || containsArtID(items, "missing") {
		t.Fatal("My Photos ownership check failed")
	}
}

func TestArtGalleryAssemblesCategoriesAndCurrentArtwork(t *testing.T) {
	request := func(_ *Config, operation string, args []string) (map[string]any, error) {
		if operation == "get_current_artwork" {
			return map[string]any{"art": map[string]any{"content_id": "mine"}}, nil
		}
		items := []artItem{{ContentID: "store", CategoryID: "MY-C0008"}}
		if len(args) == 1 && args[0] == "MY-C0002" {
			items = []artItem{{ContentID: "mine", CategoryID: "MY-C0002"}, {ContentID: "store", CategoryID: "MY-C0008"}}
		}
		raw, _ := json.Marshal(items)
		return map[string]any{"art": map[string]any{"content_list": string(raw)}}, nil
	}
	thumbnails := func(_ *Config, ids []string) (map[string]string, error) {
		if strings.Join(ids, ",") != "store,mine" {
			t.Fatalf("unexpected ids %v", ids)
		}
		return map[string]string{"store": "/cache/store.jpg", "mine": "/cache/mine.jpg"}, nil
	}
	result, err := artGalleryWith(&Config{}, request, thumbnails)
	if err != nil {
		t.Fatal(err)
	}
	items, _ := result["items"].([]map[string]any)
	if len(items) != 2 || items[0]["id"] != "store" || items[1]["id"] != "mine" || items[1]["current"] != true {
		t.Fatalf("unexpected gallery %#v", result)
	}
}

func TestArtGalleryFailureBoundaries(t *testing.T) {
	failingRequest := func(*Config, string, []string) (map[string]any, error) { return nil, errors.New("unavailable") }
	if _, err := artGalleryWith(&Config{}, failingRequest, func(*Config, []string) (map[string]string, error) { return nil, nil }); err == nil {
		t.Fatal("accepted failure of both categories")
	}

	many := make([]artItem, 101)
	for i := range many {
		many[i] = artItem{ContentID: fmt.Sprintf("item-%d", i), CategoryID: "MY-C0002"}
	}
	raw, _ := json.Marshal(many)
	request := func(_ *Config, operation string, args []string) (map[string]any, error) {
		if operation == "get_current_artwork" {
			return map[string]any{}, nil
		}
		if len(args) == 1 && args[0] == "MY-C0008" {
			return nil, errors.New("store unavailable")
		}
		return map[string]any{"art": map[string]any{"content_list": string(raw)}}, nil
	}
	if _, err := artGalleryWith(&Config{}, request, func(*Config, []string) (map[string]string, error) { return nil, nil }); err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteArtOnlyDeletesOwnedPhoto(t *testing.T) {
	request := func(*Config, string, []string) (map[string]any, error) {
		raw, _ := json.Marshal([]artItem{{ContentID: "mine", CategoryID: "MY-C0002"}, {ContentID: "store", CategoryID: "MY-C0008"}})
		return map[string]any{"art": map[string]any{"content_list": string(raw)}}, nil
	}
	called := false
	command := func(_ *Config, data map[string]any) (map[string]any, error) {
		called = true
		if data["request"] != "delete_image_list" {
			t.Fatalf("unexpected command %#v", data)
		}
		return map[string]any{"ok": true}, nil
	}
	result, err := deleteArtWith(&Config{}, "mine", request, command)
	if err != nil || !called || result["deleted_id"] != "mine" {
		t.Fatalf("result=%v called=%v err=%v", result, called, err)
	}
	called = false
	if _, err := deleteArtWith(&Config{}, "store", request, command); err == nil || called {
		t.Fatalf("store deletion: called=%v err=%v", called, err)
	}
	if _, err := deleteArtWith(&Config{}, "../bad", request, command); err == nil {
		t.Fatal("accepted invalid id")
	}
}

func TestArtRequestTransportCorrelatesResponses(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	serverDone := make(chan error, 1)
	go func() {
		op, payload := readClientFrame(t, bufio.NewReader(server))
		if op != 1 || !strings.Contains(string(payload), "art_app_request") {
			serverDone <- errors.New("invalid Art request frame")
			return
		}
		for _, response := range []map[string]any{
			{"request_id": "other", "value": "off"},
			{"request_id": "wanted", "value": "on"},
		} {
			inner, _ := json.Marshal(response)
			outer, _ := json.Marshal(map[string]any{"event": "art_app_response", "data": string(inner)})
			if _, err := server.Write(serverFrame(1, outer)); err != nil {
				serverDone <- err
				return
			}
		}
		serverDone <- nil
	}()
	if err := sendArtRequest(w, map[string]any{"request": "get_artmode_status", "request_id": "wanted"}); err != nil {
		t.Fatal(err)
	}
	got, err := waitMatchingArtResponse(w, "wanted", 2, time.Second)
	if err != nil || got["value"] != "on" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestWaitMatchingArtResponseReturnsSamsungError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		inner, _ := json.Marshal(map[string]any{"request_id": "wanted", "event": "error", "error_code": "denied"})
		outer, _ := json.Marshal(map[string]any{"data": string(inner)})
		_, _ = server.Write(serverFrame(1, outer))
	}()
	if _, err := waitMatchingArtResponse(w, "wanted", 1, time.Second); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("got %v", err)
	}
}

func TestReceiveThumbnailsRejectsUnsafeMetadata(t *testing.T) {
	if _, err := receiveThumbnails(bytes.NewReader(thumbnailFrame(t, "unexpected", []byte("x"), 0, 1)), t.TempDir(), map[string]bool{"A": true}); err == nil {
		t.Fatal("accepted unexpected id")
	}
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(70<<10))
	if _, err := receiveThumbnails(&b, t.TempDir(), map[string]bool{"A": true}); err == nil {
		t.Fatal("accepted oversized header")
	}
	metadata, err := json.Marshal(map[string]any{"fileLength": 4.5, "fileID": "A", "fileType": "jpg", "num": 0, "total": 1})
	if err != nil {
		t.Fatal(err)
	}
	b.Reset()
	_ = binary.Write(&b, binary.BigEndian, uint32(len(metadata)))
	b.Write(metadata)
	b.Write([]byte{0xff, 0xd8, 0xff, 1})
	if _, err := receiveThumbnails(&b, t.TempDir(), map[string]bool{"A": true}); err == nil {
		t.Fatal("accepted fractional thumbnail length")
	}
}

func TestReceiveThumbnailsRejectsOversizedBatch(t *testing.T) {
	b := append(thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2), thumbnailFrame(t, "B", []byte{0xff, 0xd8, 0xff, 2}, 1, 2)...)
	if _, err := receiveThumbnailsWithLimit(bytes.NewReader(b), t.TempDir(), map[string]bool{"A": true, "B": true}, 7); err == nil || !strings.Contains(err.Error(), "batch too large") {
		t.Fatalf("got %v", err)
	}
}

func TestPruneThumbnailCacheIsBoundedAndScoped(t *testing.T) {
	dir := t.TempDir()
	keep := thumbnailCachePath(dir, "keep", ".jpg")
	recent := thumbnailCachePath(dir, "recent", ".jpg")
	old := thumbnailCachePath(dir, "old", ".png")
	unrelated := filepath.Join(dir, "do-not-touch.txt")
	for _, path := range []string{keep, recent, old, unrelated} {
		if err := os.WriteFile(path, []byte("12345678"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := os.Chtimes(keep, now.Add(-3*time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(old, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := pruneThumbnailCache(dir, map[string]bool{keep: true}, 16, 2, 90*time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{keep, recent, unrelated} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
	if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale cache file remains: %v", err)
	}
}

func TestFetchArtThumbnailsFromLocalTransferEndpoint(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "state", "config.json"))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		_, err = conn.Write(thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 1))
		done <- err
	}()
	command := func(_ *Config, data map[string]any) (map[string]any, error) {
		if data["request"] != "get_thumbnail_list" {
			t.Fatalf("unexpected request %#v", data)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		return map[string]any{"art": map[string]any{"conn_info": map[string]any{"ip": "127.0.0.1", "port": float64(port), "secured": false}}}, nil
	}
	images, err := fetchArtThumbnailsWith(&Config{IP: "127.0.0.1"}, []string{"A"}, command)
	if err != nil || len(images) != 1 {
		t.Fatalf("images=%v err=%v", images, err)
	}
	if _, err := os.Stat(images["A"]); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestIntegerValue(t *testing.T) {
	for _, value := range []any{float64(1234), "4321"} {
		if _, ok := integerValue(value); !ok {
			t.Fatalf("integer parsing failed for %#v", value)
		}
	}
	for _, value := range []any{"bad", -1.0, 1.5, true} {
		if _, ok := integerValue(value); ok {
			t.Fatalf("accepted invalid integer %#v", value)
		}
	}
}
