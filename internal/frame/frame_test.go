package frame

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigRoundTripAndPermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state", "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	want := Config{IP: "192.0.2.10", Token: "private", MAC: "00:11:22:33:44:55"}
	if err := saveConfig(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(p)
		if info.Mode().Perm() != 0600 {
			t.Fatalf("mode %o", info.Mode().Perm())
		}
	}
}

func TestRejectsInvalidConfigIP(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	if err := saveConfig(Config{IP: "tv.local"}); err == nil {
		t.Fatal("expected invalid IP error")
	}
}

func TestCapabilityGroupsAndKeys(t *testing.T) {
	if len(capabilities()) < 8 {
		t.Fatal("missing documented capability groups")
	}
	if !safeKey.MatchString("KEY_MULTI_VIEW") || safeKey.MatchString("KEY_HOME;rm") {
		t.Fatal("key validation failed")
	}
}

func TestPublicConfigNeverContainsToken(t *testing.T) {
	p := Config{IP: "192.0.2.1", Token: "secret", Name: "Frame"}.public()
	if _, exists := p["token"]; exists {
		t.Fatal("public config exposed token")
	}
}

func TestCommandValidationDoesNotTouchNetwork(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	if _, err := Run([]string{"configure", "not-an-ip"}); err == nil {
		t.Fatal("expected invalid IP")
	}
	if _, err := Run([]string{"unknown"}); err == nil {
		t.Fatal("expected unknown command")
	}
}
