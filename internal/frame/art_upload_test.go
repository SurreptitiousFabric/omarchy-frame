package frame

import (
	"os"
	"path/filepath"
	"testing"
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
