package frame

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalImagePath(t *testing.T) {
	got, err := localImagePath("file:///tmp/My%20Photo.jpg")
	if err != nil || got != "/tmp/My Photo.jpg" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"relative.jpg", "https://example.com/a.jpg", "file://other-host/a.jpg"} {
		if _, err := localImagePath(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestUploadArtRejectsNonImageWithoutNetwork(t *testing.T) {
	p := filepath.Join(t.TempDir(), "not-an-image.jpg")
	if err := os.WriteFile(p, []byte("definitely not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := uploadArt(&Config{IP: "192.0.2.1"}, p); err == nil {
		t.Fatal("accepted invalid image")
	}
}

func TestOpenUploadImageAcceptsJPEGAndPNG(t *testing.T) {
	for name, data := range map[string][]byte{
		"photo.jpg": {0xff, 0xd8, 0xff, 1, 2, 3, 4, 5},
		"photo.png": {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
	} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		file, size, kind, err := openUploadImage(path)
		if err != nil || size != int64(len(data)) || kind != strings.TrimPrefix(filepath.Ext(name), ".") {
			t.Fatalf("name=%s size=%d kind=%s err=%v", name, size, kind, err)
		}
		_ = file.Close()
	}
	if _, _, _, err := openUploadImage(t.TempDir()); err == nil {
		t.Fatal("accepted directory upload")
	}
}

func TestUploadedPhotoResultReportsSelectionOutcome(t *testing.T) {
	selected := uploadedPhotoResult("mine", nil)
	if selected["selected"] != true || selected["message"] != "Photo uploaded and selected" {
		t.Fatalf("unexpected selected result %#v", selected)
	}
	partial := uploadedPhotoResult("mine", errors.New("selection unavailable"))
	if partial["selected"] != false || partial["message"] != "Photo uploaded; select it from My Photos" {
		t.Fatalf("unexpected partial result %#v", partial)
	}
}

func TestUploadArtEndToEndOverInMemoryConnections(t *testing.T) {
	image := []byte{0xff, 0xd8, 0xff, 1, 2, 3, 4, 5}
	path := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(path, image, 0600); err != nil {
		t.Fatal(err)
	}
	artClient, artServer := net.Pipe()
	uploadClient, uploadServer := net.Pipe()
	defer artServer.Close()
	defer uploadServer.Close()
	w := &wsConn{Conn: artClient, r: bufio.NewReader(artClient)}
	done := make(chan error, 1)
	go func() {
		_, payload := readClientFrame(t, bufio.NewReader(artServer))
		var envelope map[string]any
		if err := json.Unmarshal(payload, &envelope); err != nil {
			done <- err
			return
		}
		params, _ := envelope["params"].(map[string]any)
		raw, _ := params["data"].(string)
		var request map[string]any
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			done <- err
			return
		}
		requestID, _ := request["request_id"].(string)
		if request["request"] != "send_image" || requestID == "" {
			done <- fmt.Errorf("invalid upload request %#v", request)
			return
		}
		ready, _ := json.Marshal(map[string]any{"event": "ready_to_use", "request_id": requestID, "conn_info": map[string]any{"key": "transfer-key"}})
		readyEnvelope, _ := json.Marshal(map[string]any{"data": string(ready)})
		if _, err := artServer.Write(serverFrame(1, readyEnvelope)); err != nil {
			done <- err
			return
		}
		var headerLength uint32
		if err := binary.Read(uploadServer, binary.BigEndian, &headerLength); err != nil {
			done <- err
			return
		}
		headerBytes := make([]byte, headerLength)
		if _, err := io.ReadFull(uploadServer, headerBytes); err != nil {
			done <- err
			return
		}
		var header map[string]any
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			done <- err
			return
		}
		body := make([]byte, len(image))
		if _, err := io.ReadFull(uploadServer, body); err != nil {
			done <- err
			return
		}
		if header["secKey"] != "transfer-key" || header["fileType"] != "jpg" || !bytes.Equal(body, image) {
			done <- fmt.Errorf("invalid transfer header=%#v body=%x", header, body)
			return
		}
		added, _ := json.Marshal(map[string]any{"event": "image_added", "content_id": "MY-UPLOAD-1"})
		addedEnvelope, _ := json.Marshal(map[string]any{"data": string(added)})
		_, err := artServer.Write(serverFrame(1, addedEnvelope))
		done <- err
	}()
	connectArt := func(*Config, string) (*wsConn, error) { return w, nil }
	dialEndpoint := func(_ *Config, endpoint transferEndpoint, _ time.Duration) (net.Conn, error) {
		if endpoint.Key != "transfer-key" {
			t.Fatalf("unexpected endpoint %#v", endpoint)
		}
		return uploadClient, nil
	}
	selected := ""
	selectImage := func(_ *Config, id string) (map[string]any, error) {
		selected = id
		return map[string]any{"ok": true}, nil
	}
	result, err := uploadArtWith(&Config{}, path, connectArt, dialEndpoint, selectImage)
	if err != nil || result["content_id"] != "MY-UPLOAD-1" || result["selected"] != true || selected != "MY-UPLOAD-1" {
		t.Fatalf("result=%#v selected=%q err=%v", result, selected, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSetMyPhotosSlideshowPayloadValues(t *testing.T) {
	// Bounds and order are enforced at the command boundary; verify the value
	// conversion independently without contacting a TV.
	for minutes, want := range map[int]string{0: "off", 5: "5", 1440: "1440"} {
		value := "off"
		if minutes > 0 {
			value = formatSlideshowMinutes(minutes)
		}
		if value != want {
			t.Fatalf("minutes %d: got %q want %q", minutes, value, want)
		}
	}
}

func TestWaitArtEventCorrelatesAndParses(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		for _, response := range []map[string]any{
			{"event": "error", "request_id": "other", "error_code": "not_ours"},
			{"event": "ready_to_use", "request_id": "other"},
			{"event": "ready_to_use", "request_id": "wanted", "conn_info": map[string]any{"port": 1234}},
		} {
			inner, _ := json.Marshal(response)
			outer, _ := json.Marshal(map[string]any{"data": string(inner)})
			_, _ = server.Write(serverFrame(1, outer))
		}
	}()
	got, err := waitArtEvent(w, "wanted", "ready_to_use", time.Second)
	if err != nil || got.requestID != "wanted" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestWaitArtEventAllowsUncorrelatedFirmwareCompletion(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		inner, _ := json.Marshal(map[string]any{"event": "image_added", "content_id": "MY-UPLOAD-1"})
		outer, _ := json.Marshal(map[string]any{"data": string(inner)})
		_, _ = server.Write(serverFrame(1, outer))
	}()
	got, err := waitArtEvent(w, "wanted", "image_added", time.Second)
	if err != nil || got.contentID != "MY-UPLOAD-1" {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestDecodeArtResponseAcceptsObjectDataAndRejectsMalformedFields(t *testing.T) {
	message, err := json.Marshal(map[string]any{"data": map[string]any{"event": "image_added", "request_id": "wanted", "content_id": "MY-UPLOAD-1"}})
	if err != nil {
		t.Fatal(err)
	}
	got, present, err := decodeArtResponse(message)
	if err != nil || !present || got.requestID != "wanted" || got.contentID != "MY-UPLOAD-1" {
		t.Fatalf("got=%#v present=%v err=%v", got, present, err)
	}
	for _, malformed := range [][]byte{
		[]byte(`{`),
		[]byte(`{"data":"{"}`),
		[]byte(`{"data":{"request_id":12}}`),
	} {
		if _, _, err := decodeArtResponse(malformed); err == nil {
			t.Fatalf("accepted malformed response %s", malformed)
		}
	}
}

func TestWaitArtEventReturnsSamsungError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	go func() {
		inner, _ := json.Marshal(map[string]any{"event": "error", "error_code": "not_available"})
		outer, _ := json.Marshal(map[string]any{"data": string(inner)})
		_, _ = server.Write(serverFrame(1, outer))
	}()
	if _, err := waitArtEvent(w, "", "image_added", time.Second); err == nil || !strings.Contains(err.Error(), "not_available") {
		t.Fatalf("got %v", err)
	}
}

func TestConnectionInfoAcceptsObjectOrJSONString(t *testing.T) {
	for _, input := range []map[string]any{
		{"conn_info": map[string]any{"ip": "192.168.1.8", "port": float64(1234)}},
		{"conn_info": `{"ip":"192.168.1.8","port":1234}`},
	} {
		got, err := connectionInfo(input)
		if err != nil || got.IP != "192.168.1.8" || got.Port != 1234 {
			t.Fatalf("got=%v err=%v", got, err)
		}
	}
	for _, input := range []map[string]any{{}, {"conn_info": "{"}, {"conn_info": map[string]any{"port": true}}, {"conn_info": map[string]any{"secured": 1}}} {
		if _, err := connectionInfo(input); err == nil {
			t.Fatalf("accepted %#v", input)
		}
	}
}

func TestDialTVEndpointRejectsUnsafeAddresses(t *testing.T) {
	c := &Config{IP: "192.168.1.8"}
	for _, endpoint := range []transferEndpoint{
		{IP: "192.168.1.9", Port: 1234},
		{IP: "8.8.8.8", Port: 1234},
		{IP: "192.168.1.8", Port: 0},
		{IP: "not-an-ip", Port: 1234},
	} {
		if conn, err := dialTVEndpoint(c, endpoint, time.Millisecond); err == nil {
			_ = conn.Close()
			t.Fatalf("accepted %#v", endpoint)
		}
	}
}
