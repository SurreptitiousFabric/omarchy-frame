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
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(append(b, '\n'))
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpPath, p); err != nil {
		return err
	}
	keep = true
	dir, err := os.Open(filepath.Dir(p))
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func isLocalTVIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() && (ip.IsPrivate() || ip.IsLinkLocalUnicast())
}
