package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Peer struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

type Config struct {
	Peers map[string]Peer `json:"peers"`
}

func path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home dir: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	p := filepath.Join(dir, "quicksend", "config.json")
	return p, nil
}

func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Peers: make(map[string]Peer)}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Peers == nil {
		cfg.Peers = make(map[string]Peer)
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	p, err := path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func (c *Config) AddPeer(nickname, url, key string) {
	c.Peers[nickname] = Peer{URL: url, Key: key}
}

func (c *Config) RemovePeer(nickname string) bool {
	if _, ok := c.Peers[nickname]; !ok {
		return false
	}
	delete(c.Peers, nickname)
	return true
}

func (c *Config) GetPeer(nickname string) (Peer, bool) {
	p, ok := c.Peers[nickname]
	return p, ok
}
