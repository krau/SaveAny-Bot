package config

type telegramConfig struct {
	Token    string        `toml:"token" mapstructure:"token"`
	AppID    int           `toml:"app_id" mapstructure:"app_id" json:"app_id"`
	AppHash  string        `toml:"app_hash" mapstructure:"app_hash" json:"app_hash"`
	Proxy    tgProxyConfig `toml:"proxy" mapstructure:"proxy"`
	RpcRetry int           `toml:"rpc_retry" mapstructure:"rpc_retry" json:"rpc_retry"`
	Userbot  userbotConfig `toml:"userbot" mapstructure:"userbot" json:"userbot"`
}

type userbotConfig struct {
	Enable  bool   `toml:"enable" mapstructure:"enable"`
	Session string `toml:"session" mapstructure:"session"`
}

type tgProxyConfig struct {
	Enable bool   `toml:"enable" mapstructure:"enable"`
	URL    string `toml:"url" mapstructure:"url"`
}
