package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all configuration for the storage node
type Config struct {
	Node        NodeConfig        `toml:"node"`
	Coordinator CoordinatorConfig `toml:"coordinator"`
	Storage     StorageConfig     `toml:"storage"`
	API         APIConfig         `toml:"api"`
	P2P         P2PConfig         `toml:"p2p"`
}

// NodeConfig holds node identity and settings
type NodeConfig struct {
	Name         string `toml:"name"`
	DataDir      string `toml:"data_dir"`
	MaxStorageGB int    `toml:"max_storage_gb"`
	APIKey       string `toml:"api_key"`
}

// CoordinatorConfig holds coordinator connection info
type CoordinatorConfig struct {
	URL       string `toml:"url"`
	AuthToken string `toml:"auth_token"`
	PeerID    string `toml:"peer_id"`
	APIKey    string `toml:"api_key"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	ChunkDir string `toml:"chunk_dir"`
}

// APIConfig holds admin API settings
type APIConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// P2PConfig holds libp2p configuration
type P2PConfig struct {
	ListenAddresses []string `toml:"listen_addresses"`
	BootstrapPeers  []string `toml:"bootstrap_peers"`
}

// Load loads configuration from TOML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	return &config, nil
}

// Save saves configuration to TOML file
func (c *Config) Save(path string) error {
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsureDirs creates necessary directories
func (c *Config) EnsureDirs() error {
	dirs := []string{c.Node.DataDir, c.Storage.ChunkDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func (c *Config) setDefaults() {
	if c.Node.DataDir == "" {
		c.Node.DataDir = "data"
	}
	if c.Node.MaxStorageGB == 0 {
		c.Node.MaxStorageGB = 100
	}
	if c.Storage.ChunkDir == "" {
		c.Storage.ChunkDir = filepath.Join(c.Node.DataDir, "chunks")
	}
	if c.API.Host == "" {
		c.API.Host = "127.0.0.1"
	}
	if c.API.Port == 0 {
		c.API.Port = 8090
	}
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.setDefaults()
	return cfg
}
