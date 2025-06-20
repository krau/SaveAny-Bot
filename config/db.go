package config

type dbConfig struct {
	Path    string `toml:"path" mapstructure:"path"`
	Session string `toml:"session" mapstructure:"session"`
}
