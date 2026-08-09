package config

// Multi-server config, stored in the OS user config directory. Passwords go
// into the OS credential store (Windows Credential Manager, macOS Keychain,
// Linux Secret Service); the JSON file keeps one only if no keyring exists.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const keyringService = "StrikeMan"

type Server struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Password     string `json:"password,omitempty"` // keyring fallback only
	CollectionID string `json:"collectionId"`
	// Keep admin toggles across presets and map loads. Pointer so a config
	// written before this existed defaults to on rather than off.
	StickyAdmin *bool           `json:"stickyAdmin,omitempty"`
	Sticky      map[string]bool `json:"sticky,omitempty"`
	// Warmup length used by the warmup button and after every map load.
	WarmupSeconds int `json:"warmupSeconds,omitempty"`
}

const defaultWarmupSeconds = 120

func (s *Server) Warmup() int {
	if s.WarmupSeconds <= 0 {
		return defaultWarmupSeconds
	}
	return s.WarmupSeconds
}

func (s *Server) StickyEnabled() bool {
	return s.StickyAdmin == nil || *s.StickyAdmin
}

type Config struct {
	Servers []Server `json:"servers"`
	Default string   `json:"default"`
	// UI language: "en", "de", or "" to follow the operating system.
	Language string `json:"language,omitempty"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "StrikeMan", "config.json")
}

// LoadConfig reads the config, migrates the old single-server format if
// found, and fills in passwords from the keyring. A parse error is returned
// rather than swallowed, so a broken file is not mistaken for a first run.
func Load() (Config, error) {
	var cfg Config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, nil // no config yet: first run
	}
	// An editor may save the file with a UTF-8 BOM, which json rejects.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s could not be read: %w", configPath(), err)
	}

	if len(cfg.Servers) == 0 { // v0.1 format: one server at the top level
		var old Server
		if json.Unmarshal(data, &old) == nil && old.Host != "" {
			old.Name = old.Host
			cfg = Config{Servers: []Server{old}, Default: old.Name}
			cfg.Save() // moves the password into the keyring
		}
	}

	for i := range cfg.Servers {
		s := &cfg.Servers[i]
		if s.Password == "" {
			s.Password, _ = keyring.Get(keyringService, s.Name)
		}
	}
	return cfg, nil
}

// Save writes the config file and stores passwords in the keyring. Only when
// the keyring is unavailable does a password stay in the file.
func (c Config) Save() error {
	// Remember which servers existed before, to clean up renamed/removed ones.
	var prev Config
	if data, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(data, &prev)
	}

	onDisk := c
	onDisk.Servers = make([]Server, len(c.Servers))
	copy(onDisk.Servers, c.Servers)
	current := map[string]bool{}
	for i := range onDisk.Servers {
		s := &onDisk.Servers[i]
		current[s.Name] = true
		if err := keyring.Set(keyringService, s.Name, s.Password); err == nil {
			s.Password = ""
		}
	}
	for _, s := range prev.Servers {
		if !current[s.Name] {
			keyring.Delete(keyringService, s.Name)
		}
	}

	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(onDisk, "", "  ")
	return os.WriteFile(path, data, 0o600)
}

func (c Config) ServerByName(name string) *Server {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i]
		}
	}
	return nil
}
