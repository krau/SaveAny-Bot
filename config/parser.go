package config

type parserConfig struct {
	PluginEnable bool     `toml:"plugin_enable" mapstructure:"plugin_enable" json:"plugin_enable"`
	PluginDirs   []string `toml:"plugin_dirs" mapstructure:"plugin_dirs" json:"plugin_dirs"`
}
