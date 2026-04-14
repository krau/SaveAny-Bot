package config

type logConfig struct {
	Level string `toml:"level" mapstructure:"level" json:"level"`
}
