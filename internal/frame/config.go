package frame

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
)

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
	if net.ParseIP(c.IP) == nil {
		return errors.New("invalid TV IP address")
	}
	c.IP = strings.TrimSpace(c.IP)
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
