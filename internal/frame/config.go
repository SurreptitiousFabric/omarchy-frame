package frame

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var addressInError = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d{1,5})?\b`)
var tokenInError = regexp.MustCompile(`(?i)([?&]token=)[^&\s]+`)

// PublicError removes LAN endpoints and credentials from errors returned over
// the JSON process boundary. Detailed transport errors must not enter shell logs.
func PublicError(err error) string {
	if err == nil {
		return ""
	}
	s := addressInError.ReplaceAllString(err.Error(), "<local-address>")
	return tokenInError.ReplaceAllString(s, "${1}<redacted>")
}

type Config struct {
	IP    string `json:"ip"`
	Token string `json:"token,omitempty"`
	MAC   string `json:"mac,omitempty"`
	Name  string `json:"name,omitempty"`
	Model string `json:"model,omitempty"`
}

func (c Config) public() map[string]string {
	return map[string]string{"ip": c.IP, "mac": c.MAC, "name": c.Name, "model": c.Model}
}

func configPath() string {
	if p := os.Getenv("OMARCHY_FRAME_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "omarchy-frame", "config.json")
}

func loadConfig() (Config, error) {
	b, err := os.ReadFile(configPath())
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func saveConfig(c Config) error {
	c.IP = strings.TrimSpace(c.IP)
	if !isLocalTVIP(net.ParseIP(c.IP)) {
		return errors.New("TV address must be a private or link-local IP")
	}
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func isLocalTVIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() && (ip.IsPrivate() || ip.IsLinkLocalUnicast())
}
