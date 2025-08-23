package config

type parserConfig struct {
	PluginEnable bool                      `toml:"plugin_enable" mapstructure:"plugin_enable" json:"plugin_enable"`
	PluginDirs   []string                  `toml:"plugin_dirs" mapstructure:"plugin_dirs" json:"plugin_dirs"`
	Proxy        string                    `toml:"proxy" mapstructure:"proxy" json:"proxy"`
	ParserCfgs   map[string]map[string]any `mapstructure:",remain"`
}

func (c Config) GetParserConfigByName(name string) map[string]any {
	if c.Parser.ParserCfgs == nil {
		return nil
	}
	return c.Parser.ParserCfgs[name]
}
