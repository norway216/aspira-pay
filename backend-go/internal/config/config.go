// Package config loads and manages application configuration.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	NATS     NATSConfig     `yaml:"nats"`
	Engine   EngineConfig   `yaml:"engine"`
	Auth     AuthConfig     `yaml:"auth"`
	FX       FXConfig       `yaml:"fx"`
	Chain    ChainConfig    `yaml:"chain"`
}

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Port         int           `yaml:"port" default:"8080"`
	Mode         string        `yaml:"mode" default:"debug"` // debug, release, test
	ReadTimeout  time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `yaml:"write_timeout" default:"30s"`
}

// DatabaseConfig configures PostgreSQL.
type DatabaseConfig struct {
	Host     string `yaml:"host" default:"localhost"`
	Port     int    `yaml:"port" default:"5432"`
	User     string `yaml:"user" default:"aspirapay"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname" default:"aspirapay"`
	SSLMode  string `yaml:"ssl_mode" default:"disable"`
	MaxConns int    `yaml:"max_conns" default:"25"`
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig configures Redis.
type RedisConfig struct {
	Addr     string `yaml:"addr" default:"localhost:6379"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db" default:"0"`
}

// NATSConfig configures NATS JetStream.
type NATSConfig struct {
	URL      string `yaml:"url" default:"nats://localhost:4222"`
	Stream   string `yaml:"stream" default:"aspira_events"`
	Enabled  bool   `yaml:"enabled" default:"true"`
}

// EngineConfig configures the C++ engine connection.
type EngineConfig struct {
	Addr    string        `yaml:"addr" default:"localhost:9090"`
	Enabled bool          `yaml:"enabled" default:"true"`
	Timeout time.Duration `yaml:"timeout" default:"5s"`
}

// AuthConfig configures JWT authentication.
type AuthConfig struct {
	JWTSecret      string        `yaml:"jwt_secret" default:"change-me-in-production"`
	TokenExpiry    time.Duration `yaml:"token_expiry" default:"1h"`
	RefreshExpiry  time.Duration `yaml:"refresh_expiry" default:"24h"`
	Argon2Time     uint32        `yaml:"argon2_time" default:"1"`
	Argon2Memory   uint32        `yaml:"argon2_memory" default:"65536"` // 64MB
	Argon2Threads  uint8         `yaml:"argon2_threads" default:"4"`
}

// FXConfig configures the FX service.
type FXConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval" default:"60s"`
	QuoteTTL        int64         `yaml:"quote_ttl" default:"120"` // seconds
	APIURL          string        `yaml:"api_url" default:"https://api.frankfurter.app/latest?from=USD"`
}

// ChainConfig configures the blockchain audit layer.
type ChainConfig struct {
	Enabled          bool   `yaml:"enabled" default:"true"`
	Mode             string `yaml:"mode" default:"hash_chain"` // hash_chain, fabric, tendermint
	BatchSize        int    `yaml:"batch_size" default:"100"`
	BatchIntervalSec int    `yaml:"batch_interval_sec" default:"30"`
	PrivateKeyPath   string `yaml:"private_key_path"`
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read file %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: cannot parse YAML: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

// LoadEnv loads a minimal config from environment variables.
func LoadEnv() *Config {
	cfg := &Config{}
	cfg.applyDefaults()

	if v := os.Getenv("SERVER_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Server.Port)
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("ENGINE_ADDR"); v != "" {
		cfg.Engine.Addr = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}

	return cfg
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "debug"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.User == "" {
		c.Database.User = "aspirapay"
	}
	if c.Database.DBName == "" {
		c.Database.DBName = "aspirapay"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 25
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	if c.NATS.URL == "" {
		c.NATS.URL = "nats://localhost:4222"
	}
	if c.NATS.Stream == "" {
		c.NATS.Stream = "aspira_events"
	}
	if c.Engine.Addr == "" {
		c.Engine.Addr = "localhost:9090"
	}
	if c.Engine.Timeout == 0 {
		c.Engine.Timeout = 5 * time.Second
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "change-me-in-production"
	}
	if c.Auth.TokenExpiry == 0 {
		c.Auth.TokenExpiry = 1 * time.Hour
	}
	if c.Auth.RefreshExpiry == 0 {
		c.Auth.RefreshExpiry = 24 * time.Hour
	}
	if c.FX.QuoteTTL == 0 {
		c.FX.QuoteTTL = 120
	}
	if c.FX.APIURL == "" {
		c.FX.APIURL = "https://api.frankfurter.app/latest?from=USD"
	}
	if c.Chain.BatchSize == 0 {
		c.Chain.BatchSize = 100
	}
	if c.Chain.BatchIntervalSec == 0 {
		c.Chain.BatchIntervalSec = 30
	}
}
