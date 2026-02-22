package api

import (
	"fmt"
)

// Config represents the API server configuration
type Config struct {
	// Enable whether to enable the API server
	Enable bool `toml:"enable" mapstructure:"enable" json:"enable"`
	// Host is the host to bind to
	Host string `toml:"host" mapstructure:"host" json:"host"`
	// Port is the port to listen on
	Port int `toml:"port" mapstructure:"port" json:"port"`
	// Token is the authentication token for API access
	Token string `toml:"token" mapstructure:"token" json:"token"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	return nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Enable: false,
		Host:   "0.0.0.0",
		Port:   8080,
		Token:  "",
	}
}
