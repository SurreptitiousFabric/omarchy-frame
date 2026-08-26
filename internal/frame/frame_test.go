package frame

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
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
	for _, key := range []string{"KEY_HOME", "KEY_POWER", "KEY_PLAYPAUSE"} {
		if !safeRemoteKey(key, "click") {
			t.Fatalf("safe click key %q was rejected", key)
		}
	}
	for _, key := range []string{"KEY_MULTI_VIEW", "KEY_POWER"} {
		if !safeRemoteKey(key, "hold") {
			t.Fatalf("safe hold key %q was rejected", key)
		}
	}
	for _, key := range []string{"KEY_FACTORY", "KEY_RESET", "KEY_SERVICE", "KEY_HOME;rm", "KEY_UNKNOWN"} {
		if safeRemoteKey(key, "click") || safeRemoteKey(key, "hold") {
			t.Fatalf("unsafe key %q was accepted", key)
		}
	}
	if safeRemoteKey("KEY_HOME", "hold") || safeRemoteKey("KEY_POWER", "release") {
		t.Fatal("unsafe key action was accepted")
	}
}

func TestCapabilitiesCommandNeedsNoConfiguration(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "missing", "config.json"))
	result, err := Run([]string{"capabilities"})
	if err != nil {
		t.Fatal(err)
	}
	groups, ok := result["capabilities"].([]map[string]string)
	if result["ok"] != true || !ok || len(groups) != len(capabilities()) {
		t.Fatalf("unexpected capabilities result: %#v", result)
	}
	if _, err := Run([]string{"capabilities", "unexpected"}); err == nil {
		t.Fatal("capabilities accepted an argument")
	}
}

func TestCommandArgumentShapes(t *testing.T) {
	valid := [][]string{
		{"capabilities"}, {"discover"}, {"configure", "192.168.1.2"}, {"status"}, {"wake"},
		{"key", "KEY_HOME"}, {"hold", "KEY_POWER"}, {"hold", "KEY_POWER", "3000"}, {"rotate"},
		{"art", "get_artmode_status"}, {"gallery"}, {"select-art", "id"}, {"upload-art", "/tmp/photo.jpg"},
		{"delete-art", "id"}, {"slideshow", "5", "sequential"},
	}
	for _, args := range valid {
		if err := validateCommandArgs(args); err != nil {
			t.Errorf("valid shape %v: %v", args, err)
		}
	}

	invalid := [][]string{
		nil, {"unknown"}, {"capabilities", "extra"}, {"discover", "extra"}, {"configure"},
		{"status", "extra"}, {"wake", "extra"}, {"key"}, {"hold"}, {"hold", "KEY_POWER", "3000", "extra"},
		{"rotate", "extra"}, {"art"}, {"gallery", "extra"}, {"select-art"}, {"upload-art"},
		{"delete-art"}, {"slideshow", "5"},
	}
	for _, args := range invalid {
		if err := validateCommandArgs(args); err == nil {
			t.Errorf("invalid shape accepted: %v", args)
		}
	}
}

func TestQMLRemoteKeysMatchBackendAllowlist(t *testing.T) {
	keyPattern := regexp.MustCompile(`KEY_[A-Z0-9_]+`)
	used := map[string]bool{}
	names := []string{"../../BarWidget.qml", "../../Service.qml"}
	components, err := filepath.Glob("../../components/*.qml")
	if err != nil {
		t.Fatal(err)
	}
	names = append(names, components...)
	for _, name := range names {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keyPattern.FindAllString(string(body), -1) {
			used[key] = true
			if !safeRemoteKey(key, "click") && !safeRemoteKey(key, "hold") {
				t.Fatalf("QML uses key %q outside backend allowlists", key)
			}
		}
	}
	for key := range safeClickKeys {
		if !used[key] {
			t.Fatalf("backend permits unused click key %q", key)
		}
	}
}

func TestGalleryCardsRemainKeyboardAccessible(t *testing.T) {
	body, err := os.ReadFile("../../components/GalleryCard.qml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	for _, contract := range []string{
		"activeFocusOnTab: true",
		"Accessible.role: Accessible.Button",
		"Keys.onReturnPressed",
		"Keys.onEnterPressed",
		"Keys.onSpacePressed",
		"card.forceActiveFocus()",
		"focusable: true",
	} {
		if !strings.Contains(source, contract) {
			t.Fatalf("GalleryCard keyboard contract missing %q", contract)
		}
	}
}

func TestHeldKeyAlwaysAttemptsRelease(t *testing.T) {
	var actions []string
	failedRelease := true
	err := sendHeldKey(func(action string) error {
		actions = append(actions, action)
		if action == "Release" && failedRelease {
			failedRelease = false
			return errors.New("temporary release failure")
		}
		return nil
	}, 3*time.Second, func(duration time.Duration) {
		if duration != 3*time.Second {
			t.Fatalf("unexpected hold duration %v", duration)
		}
	})
	if err != nil || strings.Join(actions, ",") != "Press,Release,Release" {
		t.Fatalf("actions=%v err=%v", actions, err)
	}

	actions = nil
	err = sendHeldKey(func(action string) error {
		actions = append(actions, action)
		return errors.New("press failed")
	}, time.Second, func(time.Duration) { t.Fatal("waited after failed press") })
	if err == nil || strings.Join(actions, ",") != "Press" {
		t.Fatalf("actions=%v err=%v", actions, err)
	}
}

func TestPublicConfigNeverContainsToken(t *testing.T) {
	p := Config{IP: "192.168.1.2", Token: "secret", Name: "Frame"}.public()
	if _, exists := p["token"]; exists {
		t.Fatal("public config exposed token")
	}
}

func TestGetInfoWithClientValidatesResponses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"device":{"name":"Frame","modelName":"QE55LS03B"},"version":"2.0"}`))
	})
	mux.HandleFunc("/bad-status", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusForbidden) })
	mux.HandleFunc("/bad-json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) })
	mux.HandleFunc("/large", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(strings.Repeat(" ", (1<<20)+1))) })
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/ok", http.StatusFound) })
	server := httptest.NewServer(mux)
	defer server.Close()
	client := lanHTTPClient(time.Second)

	got, err := getInfoWithClient("192.168.1.8", server.URL+"/ok", client)
	if err != nil || got.Device.Name != "Frame" || got.Device.ModelName != "QE55LS03B" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	for _, path := range []string{"/bad-status", "/bad-json", "/large", "/redirect"} {
		if _, err := getInfoWithClient("192.168.1.8", server.URL+path, client); err == nil {
			t.Fatalf("accepted %s response", path)
		}
	}
	if _, err := getInfoWithClient("127.0.0.1", server.URL+"/ok", client); err == nil {
		t.Fatal("accepted loopback TV address")
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
	for _, args := range [][]string{nil, {"configure"}, {"configure", "192.168.1.2", "extra"}, {"discover", "extra"}, {"status", "extra"}, {"wake", "extra"}, {"key"}, {"key", "KEY_HOME", "extra"}, {"hold"}, {"hold", "KEY_POWER", "3000", "extra"}, {"rotate", "extra"}, {"art"}, {"gallery", "extra"}, {"select-art"}} {
		if _, err := Run(args); err == nil {
			t.Fatalf("expected validation error for %v", args)
		}
	}
}

func TestConfigureValidationCanRecoverFromMalformedConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	if err := os.WriteFile(p, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Run([]string{"configure", "not-an-ip"})
	if err == nil || !strings.Contains(err.Error(), "private or link-local") {
		t.Fatalf("configure was blocked by malformed config: %v", err)
	}
}

func TestConcurrentConfigWritesRemainValid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state", "config.json")
	t.Setenv("OMARCHY_FRAME_CONFIG", p)
	const writers = 20
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 1; i <= writers; i++ {
		wg.Add(1)
		go func(lastOctet int) {
			defer wg.Done()
			errs <- saveConfig(Config{IP: fmt.Sprintf("192.168.44.%d", lastOctet), Name: fmt.Sprintf("Frame %d", lastOctet)})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := loadConfig()
	if err != nil || !strings.HasPrefix(got.IP, "192.168.44.") {
		t.Fatalf("invalid final config %#v: %v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(p), ".config-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, %v", matches, err)
	}
}

func TestConfiguredCommandArgumentValidation(t *testing.T) {
	t.Setenv("OMARCHY_FRAME_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	if err := saveConfig(Config{IP: "192.168.9.9"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"status", "extra"},
		{"key"}, {"key", "KEY_FACTORY"}, {"key", "KEY_HOME", "extra"},
		{"hold"}, {"hold", "KEY_HOME", "100", "extra"}, {"hold", "KEY_HOME", "100"}, {"hold", "KEY_HOME", "99"}, {"hold", "KEY_HOME", "10001"},
		{"wake", "extra"}, {"rotate", "extra"}, {"gallery", "extra"},
		{"art"}, {"art", "set_artmode_status"}, {"art", "get_content_list"}, {"art", "get_artmode_status", "unexpected"},
		{"select-art"}, {"select-art", "../bad"},
		{"upload-art"}, {"upload-art", "relative.jpg"},
		{"delete-art"}, {"delete-art", "../bad"},
		{"slideshow"}, {"slideshow", "bad", "sequential"}, {"slideshow", "1441", "shuffle"}, {"slideshow", "5", "random"},
		{"unknown"},
	} {
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
	for _, id := range []string{"SAM-F0206", "MY_F0001"} {
		if !safeArtID.MatchString(id) {
			t.Fatalf("rejected %q", id)
		}
	}
	for _, id := range []string{"", "../bad", "bad?query", "bad space"} {
		if safeArtID.MatchString(id) {
			t.Fatalf("accepted %q", id)
		}
	}
}

func TestArtRequestAllowlist(t *testing.T) {
	for _, request := range []string{"get_artmode_status", "get_current_artwork", "get_category_list", "get_content_list", "get_slideshow_status"} {
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
	if _, err := artRequest(&Config{}, "get_content_list", []string{"../bad"}); err == nil {
		t.Fatal("accepted invalid category")
	}
}

func TestArtRequestWrapperEndToEnd(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	w := &wsConn{Conn: client, r: bufio.NewReader(client)}
	done := make(chan error, 1)
	go func() {
		_, payload := readClientFrame(t, bufio.NewReader(server))
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
		response, _ := json.Marshal(map[string]any{"request_id": requestID, "event": "get_artmode_status", "value": "on"})
		responseEnvelope, _ := json.Marshal(map[string]any{"data": string(response)})
		_, err := server.Write(serverFrame(1, responseEnvelope))
		done <- err
	}()
	connectArt := func(_ *Config, channel string, timeout time.Duration) (*wsConn, error) {
		if channel != "com.samsung.art-app" || timeout != 2*time.Second {
			t.Fatalf("channel=%q timeout=%v", channel, timeout)
		}
		return w, nil
	}
	result, err := artRequestWithConnector(&Config{}, "get_artmode_status", nil, 2*time.Second, time.Second, connectArt)
	if err != nil || artModeFromResult(result) != "art" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestArtRequestID(t *testing.T) {
	a, err := artRequestID()
	if err != nil {
		t.Fatal(err)
	}
	b, err := artRequestID()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 36 || a == b || strings.Count(a, "-") != 4 {
		t.Fatalf("bad IDs %q %q", a, b)
	}
}

func TestArtModeFromResult(t *testing.T) {
	tests := []struct {
		name   string
		result map[string]any
		want   string
	}{
		{"art", map[string]any{"art": map[string]any{"event": "get_artmode_status", "value": "on"}}, "art"},
		{"art navigation", map[string]any{"art": map[string]any{"event": "get_artmode_status", "value": "nav"}}, "art"},
		{"tv", map[string]any{"art": map[string]any{"event": "artmode_status", "value": " OFF "}}, "tv"},
		{"missing art payload", map[string]any{"ok": true}, "unknown"},
		{"missing value", map[string]any{"art": map[string]any{"event": "artmode_status"}}, "unknown"},
		{"unexpected value", map[string]any{"art": map[string]any{"value": "standby"}}, "unknown"},
		{"wrong payload type", map[string]any{"art": "on"}, "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := artModeFromResult(test.result); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFrameStatusModesAndFallback(t *testing.T) {
	config := &Config{IP: "192.168.1.20", Name: "Frame"}
	infoFor := func(model string) func(string) (APIInfo, error) {
		return func(string) (APIInfo, error) {
			var result APIInfo
			result.Device.PowerState = "standby"
			result.Device.ModelName = model
			return result, nil
		}
	}

	for _, test := range []struct {
		name   string
		art    map[string]any
		artErr error
		model  string
		want   string
	}{
		{"art", map[string]any{"art": map[string]any{"value": "on"}}, nil, "QE55LS03B", "art"},
		{"reliable tv", map[string]any{"art": map[string]any{"value": "off"}}, nil, "QE55LS03D", "tv"},
		{"ambiguous 2022 off", map[string]any{"art": map[string]any{"value": "off"}}, nil, "QE55LS03BAUXXH", "unknown"},
		{"unsupported Art service", nil, errors.New("unavailable"), "QE55LS03D", "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := frameStatus(config, infoFor(test.model), func(*Config) (map[string]any, error) { return test.art, test.artErr })
			if got["online"] != true || got["mode"] != test.want || got["power"] != "standby" {
				t.Fatalf("unexpected status %#v", got)
			}
		})
	}

	artCalled := false
	offline := frameStatus(config, func(string) (APIInfo, error) { return APIInfo{}, errors.New("offline") }, func(*Config) (map[string]any, error) {
		artCalled = true
		return nil, nil
	})
	if offline["online"] != false || offline["mode"] != "offline" || artCalled {
		t.Fatalf("unexpected offline status %#v, artCalled=%v", offline, artCalled)
	}
}

func TestMatchingArtResponse(t *testing.T) {
	envelope := func(data any) []byte {
		b, err := json.Marshal(map[string]any{"event": "art_app_response", "data": data})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	inner := func(values map[string]any) string {
		b, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	if _, matched, err := matchingArtResponse([]byte(`{"event":"ms.channel.ready"}`), "wanted"); err != nil || matched {
		t.Fatalf("ready event matched: matched=%v err=%v", matched, err)
	}
	if _, matched, err := matchingArtResponse(envelope(inner(map[string]any{"event": "art_mode_changed", "value": "on"})), "wanted"); err != nil || matched {
		t.Fatalf("unsolicited event matched: matched=%v err=%v", matched, err)
	}
	if _, matched, err := matchingArtResponse(envelope(inner(map[string]any{"request_id": "other", "value": "on"})), "wanted"); err != nil || matched {
		t.Fatalf("wrong request matched: matched=%v err=%v", matched, err)
	}
	got, matched, err := matchingArtResponse(envelope(inner(map[string]any{"request_id": "wanted", "value": "on"})), "wanted")
	if err != nil || !matched || got["value"] != "on" {
		t.Fatalf("matching response failed: got=%v matched=%v err=%v", got, matched, err)
	}
	if _, matched, err = matchingArtResponse(envelope(inner(map[string]any{"id": "wanted", "event": "error", "error_code": "denied"})), "wanted"); err == nil || !matched {
		t.Fatalf("matching error was not returned: matched=%v err=%v", matched, err)
	}
}

func TestAmbiguousArtOffModel(t *testing.T) {
	for _, model := range []string{"QE55LS03BAUXXH", "qn65ls03bafxza", "LS03B"} {
		if !ambiguousArtOffModel(model) {
			t.Fatalf("expected %q to be ambiguous", model)
		}
	}
	if ambiguousArtOffModel("QE55LS03DAUXXH") {
		t.Fatal("newer model should retain its reported TV state")
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
