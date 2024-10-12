package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Temp     tempConfig     `toml:"temp" mapstructure:"temp"`
	Log      logConfig      `toml:"log" mapstructure:"log"`
	DB       dbConfig       `toml:"db" mapstructure:"db"`
	Telegram telegramConfig `toml:"telegram" mapstructure:"telegram"`
	Storage  storageConfig  `toml:"storage" mapstructure:"storage"`
}

type tempConfig struct {
	BasePath string `toml:"base_path" mapstructure:"base_path"`
	CacheTTL int64  `toml:"cache_ttl" mapstructure:"cache_ttl"`
}

type logConfig struct {
	Level       string `toml:"level" mapstructure:"level"`
	File        string `toml:"file" mapstructure:"file"`
	BackupCount uint   `toml:"backup_count" mapstructure:"backup_count"`
}

type dbConfig struct {
	Path string `toml:"path" mapstructure:"path"`
}

type telegramConfig struct {
	Token   string  `toml:"token" mapstructure:"token"`
	AppID   int32   `toml:"app_id" mapstructure:"app_id"`
	AppHash string  `toml:"app_hash" mapstructure:"app_hash"`
	Admins  []int64 `toml:"admins" mapstructure:"admins"`
}

type storageConfig struct {
	Alist  alistConfig  `toml:"alist" mapstructure:"alist"`
	Local  localConfig  `toml:"local" mapstructure:"local"`
	Webdav webdavConfig `toml:"webdav" mapstructure:"webdav"`
}

type alistConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	URL      string `toml:"url" mapstructure:"url"`
	Username string `toml:"username" mapstructure:"username"`
	Password string `toml:"password" mapstructure:"password"`
	BasePath string `toml:"base_path" mapstructure:"base_path"`
	TokenExp int64  `toml:"token_exp" mapstructure:"token_exp"`
}

type localConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	BasePath string `toml:"base_path" mapstructure:"base_path"`
}

type webdavConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	URL      string `toml:"url" mapstructure:"url"`
	Username string `toml:"username" mapstructure:"username"`
	Password string `toml:"password" mapstructure:"password"`
	BasePath string `toml:"base_path" mapstructure:"base_path"`
}

var Cfg *Config

func Init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("toml")

	viper.SetDefault("temp.base_path", "cache/")
	viper.SetDefault("temp.cache_ttl", 3600)

	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.file", "logs/saveany.log")
	viper.SetDefault("log.backup_count", 7)

	viper.SetDefault("db.path", "data/saveany.db")

	viper.SetDefault("telegram.api", "https://api.telegram.org")

	viper.SetDefault("storage.alist.base_path", "/")
	viper.SetDefault("storage.alist.token_exp", 3600)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file, ", err)
		os.Exit(1)
	}

	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		fmt.Println("Error unmarshalling config file, ", err)
		os.Exit(1)
	}
}
