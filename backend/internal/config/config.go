package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type DatabaseConfig struct {
	Provider    string `yaml:"provider"`
	TableName   string `yaml:"table_name"`
	Region      string `yaml:"region"`
	EndpointURL string `yaml:"endpoint_url,omitempty"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type AuthConfig struct {
	JWTSecret   string        `yaml:"jwt_secret"`
	TokenExpiry time.Duration `yaml:"token_expiry"`
	RequireAuth bool          `yaml:"require_auth"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func LoadConfig() (*Config, error) {
	env := os.Getenv("TIMEDB_ENV")
	if env == "" {
		env = "local"
	}

	configPath := fmt.Sprintf("config/environments/%s.yaml", env)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Auth.TokenExpiry == 0 {
		cfg.Auth.TokenExpiry = 24 * time.Hour
	}

	return &cfg, nil
}
