package frame

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigRoundTripAndPermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state", "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	want := Config{IP: "192.168.50.10", Token: "private", MAC: "00:11:22:33:44:55"}
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
	if err := saveConfig(Config{IP: "8.8.8.8"}); err == nil {
		t.Fatal("expected public IP rejection")
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
	p := Config{IP: "192.168.1.2", Token: "secret", Name: "Frame"}.public()
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
	for _, args := range [][]string{nil, {"configure"}, {"key"}, {"hold"}, {"launch"}, {"art"}} {
		if _, err := Run(args); err == nil {
			t.Fatalf("expected validation error for %v", args)
		}
	}
}

func TestConfiguredCommandArgumentValidation(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	if err := saveConfig(Config{IP: "192.168.9.9"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"key"}, {"hold"}, {"launch"}, {"art"}, {"hold", "KEY_HOME", "99"}, {"hold", "KEY_HOME", "10001"}, {"launch", "../bad"}, {"art", "set_artmode_status"}, {"unknown"}} {
		if _, err := Run(args); err == nil {
			t.Fatalf("expected validation error for %v", args)
		}
	}
}

func TestCommandRejectsTamperedPublicAddress(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	b, _ := json.Marshal(Config{IP: "8.8.8.8"})
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run([]string{"status"}); err == nil {
		t.Fatal("accepted public configured address")
	}
}

func TestConfigMissingMalformedAndTrimmed(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	if got, err := loadConfig(); err != nil || got != (Config{}) {
		t.Fatalf("missing: %#v %v", got, err)
	}
	if err := os.WriteFile(p, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected malformed JSON error")
	}
	if err := saveConfig(Config{IP: " 192.168.2.3 "}); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig()
	if err != nil || got.IP != "192.168.2.3" {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestWakePacketAndAppIDValidation(t *testing.T) {
	p, err := wakePacket("00:11:22:33:44:55")
	if err != nil || len(p) != 102 {
		t.Fatalf("packet len=%d err=%v", len(p), err)
	}
	if string(p[:6]) != string([]byte{255, 255, 255, 255, 255, 255}) {
		t.Fatal("bad magic prefix")
	}
	if _, err := wakePacket("bad"); err == nil {
		t.Fatal("accepted bad MAC")
	}
	for _, id := range []string{"org.tizen.netflix-app", "111299001912"} {
		if !safeAppID.MatchString(id) {
			t.Fatalf("rejected %q", id)
		}
	}
	for _, id := range []string{"", "../bad", "bad?query", "bad space"} {
		if safeAppID.MatchString(id) {
			t.Fatalf("accepted %q", id)
		}
	}
}

func TestArtRequestAllowlist(t *testing.T) {
	for _, request := range []string{"get_artmode_status", "get_current_artwork", "get_category_list", "get_slideshow_status"} {
		if !safeArtRequests[request] {
			t.Fatalf("missing %q", request)
		}
	}
	if _, err := artRequest(&Config{}, "set_artmode_status", nil); err == nil {
		t.Fatal("accepted write request")
	}
	if _, err := artRequest(&Config{}, "get_artmode_status", []string{"unexpected"}); err == nil {
		t.Fatal("accepted arguments")
	}
}

func TestPublicErrorRedactsEndpointsAndTokens(t *testing.T) {
	got := PublicError(errors.New("read tcp 192.168.1.4:1234->10.0.0.2:8002 wss://host/x?token=secret&name=ok"))
	for _, secret := range []string{"192.168.1.4", "10.0.0.2", "secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("leaked %q in %q", secret, got)
		}
	}
	if PublicError(nil) != "" {
		t.Fatal("nil error should be empty")
	}
}
