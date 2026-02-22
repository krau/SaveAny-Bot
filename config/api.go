package config

type apiConfig struct {
	Enable bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	Host   string `toml:"host" mapstructure:"host" json:"host"`
	Port   int    `toml:"port" mapstructure:"port" json:"port"`
	Token  string `toml:"token" mapstructure:"token" json:"token"`
}
