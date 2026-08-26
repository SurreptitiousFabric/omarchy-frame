package frame

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const maxConfigSize = 64 << 10

var addressInError = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d{1,5})?\b`)
var bracketedIPv6InError = regexp.MustCompile(`\[[0-9A-Fa-f:]+(?:%[0-9A-Za-z_.-]+)?\](?::\d{1,5})?`)
var tokenInError = regexp.MustCompile(`(?i)([?&]token=)[^&\s]+`)

// PublicError removes LAN endpoints and credentials from errors returned over
// the JSON process boundary. Detailed transport errors must not enter shell logs.
func PublicError(err error) string {
	if err == nil {
		return ""
	}
	s := addressInError.ReplaceAllString(err.Error(), "<local-address>")
	s = bracketedIPv6InError.ReplaceAllString(s, "<local-address>")
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
	return map[string]string{"name": c.Name, "model": c.Model}
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
	return loadConfigAt(configPath())
}

func loadConfigAt(p string) (Config, error) {
	if err := requirePrivateStateDir(filepath.Dir(p)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	info, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Config{}, errors.New("TV state must be a regular file")
	}
	if info.Mode().Perm()&0077 != 0 {
		return Config{}, errors.New("TV state permissions must be owner-only")
	}
	if info.Size() > maxConfigSize {
		return Config{}, errors.New("TV state file is too large")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	if c.Token != "" && !validPairingToken(c.Token) {
		return Config{}, errors.New("TV state contains an invalid pairing token")
	}
	return c, nil
}

func saveConfig(c Config) error {
	c.IP = strings.TrimSpace(c.IP)
	if !isLocalTVIP(net.ParseIP(c.IP)) {
		return errors.New("TV address must be a private or link-local IP")
	}
	if c.Token != "" && !validPairingToken(c.Token) {
		return errors.New("TV pairing token is invalid")
	}
	p := configPath()
	return withConfigLock(p, func() error { return saveConfigUnlocked(p, c) })
}

func saveConfigUnlocked(p string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > maxConfigSize {
		return errors.New("TV state is too large")
	}
	dirPath := filepath.Dir(p)
	tmp, err := os.CreateTemp(dirPath, ".config-*.tmp")
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
	dir, err := os.Open(dirPath)
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

func withConfigLock(p string, fn func() error) error {
	dirPath := filepath.Dir(p)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}
	if err := requirePrivateStateDir(dirPath); err != nil {
		return err
	}
	lockPath := filepath.Join(dirPath, ".config.lock")
	if info, err := os.Lstat(lockPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("TV state lock must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	lockInfo, err := lock.Stat()
	if err != nil {
		return err
	}
	if lockInfo.Mode().Perm()&0077 != 0 {
		return errors.New("TV state lock permissions must be owner-only")
	}
	if stat, ok := lockInfo.Sys().(*syscall.Stat_t); ok && stat.Nlink != 1 {
		return errors.New("TV state lock must not have multiple links")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	return fn()
}

func persistPairingToken(c *Config, token string) error {
	if !validPairingToken(token) {
		return errors.New("TV returned an invalid pairing token")
	}
	p := configPath()
	return withConfigLock(p, func() error {
		current, err := loadConfigAt(p)
		if err != nil {
			return err
		}
		if current.IP == "" {
			current = *c
		} else if currentIP, pairingIP := net.ParseIP(current.IP), net.ParseIP(c.IP); currentIP == nil || pairingIP == nil || !currentIP.Equal(pairingIP) {
			return errors.New("TV configuration changed during pairing; retry the command")
		}
		current.Token = token
		if err := saveConfigUnlocked(p, current); err != nil {
			return err
		}
		*c = current
		return nil
	})
}

func validPairingToken(token string) bool {
	if len(token) == 0 || len(token) > 4096 {
		return false
	}
	for _, r := range token {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

func requirePrivateStateDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("TV state parent must be a directory")
	}
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("TV state directory permissions must be owner-only")
	}
	return nil
}

func isLocalTVIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() && (ip.IsPrivate() || ip.IsLinkLocalUnicast())
}
