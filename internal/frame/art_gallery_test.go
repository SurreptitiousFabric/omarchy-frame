package frame

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
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

func TestReceiveThumbnailsAllowsCleanPartialBatch(t *testing.T) {
	b := thumbnailFrame(t, "A", []byte{0xff, 0xd8, 0xff, 1}, 0, 2)
	got, err := receiveThumbnails(bytes.NewReader(b), t.TempDir(), map[string]bool{"A": true, "B": true})
	if err != nil || len(got) != 1 {
		t.Fatalf("got=%v err=%v", got, err)
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

func TestReceiveThumbnailsRejectsUnsafeMetadata(t *testing.T) {
	if _, err := receiveThumbnails(bytes.NewReader(thumbnailFrame(t, "unexpected", []byte("x"), 0, 1)), t.TempDir(), map[string]bool{"A": true}); err == nil {
		t.Fatal("accepted unexpected id")
	}
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(70<<10))
	if _, err := receiveThumbnails(&b, t.TempDir(), map[string]bool{"A": true}); err == nil {
		t.Fatal("accepted oversized header")
	}
}

func TestPortNumber(t *testing.T) {
	if portNumber(float64(1234)) != 1234 || portNumber("4321") != 4321 || portNumber("bad") != 0 {
		t.Fatal("port parsing failed")
	}
	if !boolValue(true) || !boolValue("true") || boolValue("false") {
		t.Fatal("boolean parsing failed")
	}
}
