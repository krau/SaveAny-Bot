package config

type dbConfig struct {
	Session string `toml:"session" mapstructure:"session" json:"session"`
}
