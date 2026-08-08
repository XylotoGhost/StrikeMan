package main

// Config is stored outside the repo in the OS user config directory,
// so the RCON password never ends up in git.

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Password     string `json:"password"`
	CollectionID string `json:"collectionId"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "StrikeMan", "config.json")
}

func LoadConfig() Config {
	cfg := Config{Host: "127.0.0.1", Port: 27015}
	data, err := os.ReadFile(configPath())
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

func (c Config) Save() error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, data, 0o600)
}
