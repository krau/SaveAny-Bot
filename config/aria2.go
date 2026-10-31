package config

type aria2Config struct {
	Enable   bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	Url      string `toml:"url" mapstructure:"url" json:"url"`
	Secret   string `toml:"secret" mapstructure:"secret" json:"secret"`
	KeepFile bool   `toml:"keep_file" mapstructure:"keep_file" json:"keep_file"`
}
