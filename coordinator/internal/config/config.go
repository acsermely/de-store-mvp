package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all configuration for the coordinator
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	P2P      P2PConfig      `toml:"p2p"`
	Storage  StorageConfig  `toml:"storage"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	ReadTimeout  int    `toml:"read_timeout"`
	WriteTimeout int    `toml:"write_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	SSLMode  string `toml:"ssl_mode"`
}

// P2PConfig holds libp2p configuration
type P2PConfig struct {
	ListenAddresses []string `toml:"listen_addresses"`
	BootstrapPeers  []string `toml:"bootstrap_peers"`
	EnableQUIC      bool     `toml:"enable_quic"`
	EnableTCP       bool     `toml:"enable_tcp"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	ChunkSizeBytes          int64 `toml:"chunk_size_bytes"`
	DefaultReplicas         int   `toml:"default_replicas"`
	ProofDifficulty         int   `toml:"proof_difficulty"`
	ProofIntervalHours      int   `toml:"proof_interval_hours"`
	StorageCreditPerGBMonth int64 `toml:"storage_credit_per_gb_month"`
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
	config.SetDefaults()

	return &config, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.SetDefaults()
	return cfg
}

// DatabaseURL returns the PostgreSQL connection URL
func (c *DatabaseConfig) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// SetDefaults sets default values for config
func (c *Config) SetDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.User == "" {
		c.Database.User = "postgres"
	}
	if c.Database.Database == "" {
		c.Database.Database = "coordinator"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.P2P.EnableTCP == false && c.P2P.EnableQUIC == false {
		c.P2P.EnableTCP = true
		c.P2P.EnableQUIC = true
	}
	if c.Storage.ChunkSizeBytes == 0 {
		c.Storage.ChunkSizeBytes = 256 * 1024 // 256KB
	}
	if c.Storage.DefaultReplicas == 0 {
		c.Storage.DefaultReplicas = 3
	}
	if c.Storage.ProofDifficulty == 0 {
		c.Storage.ProofDifficulty = 1000
	}
	if c.Storage.ProofIntervalHours == 0 {
		c.Storage.ProofIntervalHours = 4
	}
	if c.Storage.StorageCreditPerGBMonth == 0 {
		c.Storage.StorageCreditPerGBMonth = 100 // 100 credits per GB per month
	}
}
