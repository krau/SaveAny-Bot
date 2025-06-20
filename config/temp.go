package config

type tempConfig struct {
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}
