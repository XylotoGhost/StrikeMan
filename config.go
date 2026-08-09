package main

// Multi-server config, stored in the OS user config directory. Passwords go
// into the OS credential store (Windows Credential Manager, macOS Keychain,
// Linux Secret Service); the JSON file keeps one only if no keyring exists.

import (
	"encoding/json"
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
}

type Config struct {
	Servers []Server `json:"servers"`
	Default string   `json:"default"`
	GsiPort int      `json:"gsiPort"` // 0 disables the live-score listener
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "StrikeMan", "config.json")
}

// LoadConfig reads the config, migrates the old single-server format if
// found, and fills in passwords from the keyring.
func LoadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)

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
	return cfg
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

func (c Config) serverByName(name string) *Server {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i]
		}
	}
	return nil
}
