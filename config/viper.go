package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/datatypes"
)

type Config struct {
	Workers      int  `toml:"workers" mapstructure:"workers"`
	Retry        int  `toml:"retry" mapstructure:"retry"`
	NoCleanCache bool `toml:"no_clean_cache" mapstructure:"no_clean_cache"`

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
	Token   string `toml:"token" mapstructure:"token"`
	AppID   int    `toml:"app_id" mapstructure:"app_id"`
	AppHash string `toml:"app_hash" mapstructure:"app_hash"`
	// 白名单用户
	Admins []int64     `toml:"admins" mapstructure:"admins"` // Whitelisted users
	Proxy  proxyConfig `toml:"proxy" mapstructure:"proxy"`
}

type proxyConfig struct {
	Enable bool   `toml:"enable" mapstructure:"enable"`
	URL    string `toml:"url" mapstructure:"url"`
}

// pre-defined storages, for compatibility.
/*
在配置文件中定义的存储将会为telegram.admins中的每个用户创建一个存储模型
*/
// these config will be removed in the future.
type storageConfig struct {
	Alist  AlistConfig  `toml:"alist" mapstructure:"alist"`
	Local  LocalConfig  `toml:"local" mapstructure:"local"`
	Webdav WebdavConfig `toml:"webdav" mapstructure:"webdav"`
}

type AlistConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	URL      string `toml:"url" mapstructure:"url"`
	Username string `toml:"username" mapstructure:"username"`
	Password string `toml:"password" mapstructure:"password"`
	Token    string `toml:"token" mapstructure:"token"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	TokenExp int64  `toml:"token_exp" mapstructure:"token_exp" json:"token_exp"`
}

func (a *AlistConfig) ToJSON() datatypes.JSON {
	tokenExp := strconv.FormatInt(a.TokenExp, 10)
	return datatypes.JSON([]byte(`{"url":"` + a.URL + `","username":"` + a.Username + `","password":"` + a.Password + `","token":"` + a.Token + `","base_path":"` + a.BasePath + `","token_exp":` + tokenExp + `}`))
}

type LocalConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	BasePath string `toml:"base_path" mapstructure:"base_path"`
}

func (l *LocalConfig) ToJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{"base_path":"` + l.BasePath + `"}`))
}

type WebdavConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable"`
	URL      string `toml:"url" mapstructure:"url"`
	Username string `toml:"username" mapstructure:"username"`
	Password string `toml:"password" mapstructure:"password"`
	BasePath string `toml:"base_path" mapstructure:"base_path"`
}

func (w *WebdavConfig) ToJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{"url":"` + w.URL + `","username":"` + w.Username + `","password":"` + w.Password + `","base_path":"` + w.BasePath + `"}`))
}

var Cfg *Config

func Init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/saveany/")
	viper.SetConfigType("toml")
	viper.SetEnvPrefix("SAVEANY")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetDefault("workers", 3)
	viper.SetDefault("retry", 3)

	viper.SetDefault("telegram.app_id", 1025907)
	viper.SetDefault("telegram.app_hash", "452b0359b988148995f22ff0f4229750")

	viper.SetDefault("temp.base_path", "cache/")
	viper.SetDefault("temp.cache_ttl", 3600)

	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.file", "logs/saveany.log")
	viper.SetDefault("log.backup_count", 7)

	viper.SetDefault("db.path", "data/saveany.db")

	viper.SafeWriteConfigAs("config.toml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file, ", err)
		os.Exit(1)
	}

	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		fmt.Println("Error unmarshalling config file, ", err)
		os.Exit(1)
	}
	if Cfg.Storage != (storageConfig{}) {
		fmt.Println("警告: 存储配置已经废弃, 未来版本将会移除.\n请直接使用 Bot 命令添加存储.")
	}
	if Cfg.Workers < 1 || Cfg.Retry < 1 {
		fmt.Println("Invalid workers or retry value")
		os.Exit(1)
	}
}

func Set(key string, value any) {
	viper.Set(key, value)
}

func ReloadConfig() error {
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if error := viper.Unmarshal(Cfg); error != nil {
		return error
	}
	return nil
}
