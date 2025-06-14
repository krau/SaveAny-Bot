package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config/storage"
	"github.com/spf13/viper"
)

type Config struct {
	Lang         string `toml:"lang" mapstructure:"lang" json:"lang"`
	Workers      int    `toml:"workers" mapstructure:"workers"`
	Retry        int    `toml:"retry" mapstructure:"retry"`
	NoCleanCache bool   `toml:"no_clean_cache" mapstructure:"no_clean_cache" json:"no_clean_cache"`
	Threads      int    `toml:"threads" mapstructure:"threads" json:"threads"`
	Stream       bool   `toml:"stream" mapstructure:"stream" json:"stream"`

	Users []userConfig `toml:"users" mapstructure:"users" json:"users"`

	Temp     tempConfig              `toml:"temp" mapstructure:"temp"`
	Log      logConfig               `toml:"log" mapstructure:"log"`
	DB       dbConfig                `toml:"db" mapstructure:"db"`
	Telegram telegramConfig          `toml:"telegram" mapstructure:"telegram"`
	Storages []storage.StorageConfig `toml:"-" mapstructure:"-" json:"storages"`
}

type tempConfig struct {
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	CacheTTL int64  `toml:"cache_ttl" mapstructure:"cache_ttl" json:"cache_ttl"`
}

type logConfig struct {
	Level       string `toml:"level" mapstructure:"level"`
	File        string `toml:"file" mapstructure:"file"`
	BackupCount uint   `toml:"backup_count" mapstructure:"backup_count" json:"backup_count"`
}

type dbConfig struct {
	Path    string `toml:"path" mapstructure:"path"`
	Session string `toml:"session" mapstructure:"session"`
	Expire  int64  `toml:"expire" mapstructure:"expire"`
}

type telegramConfig struct {
	Token      string        `toml:"token" mapstructure:"token"`
	AppID      int           `toml:"app_id" mapstructure:"app_id" json:"app_id"`
	AppHash    string        `toml:"app_hash" mapstructure:"app_hash" json:"app_hash"`
	Timeout    int           `toml:"timeout" mapstructure:"timeout" json:"timeout"`
	Proxy      proxyConfig   `toml:"proxy" mapstructure:"proxy"`
	RpcRetry   int           `toml:"rpc_retry" mapstructure:"rpc_retry" json:"rpc_retry"`
	Userbot    userbotConfig `toml:"userbot" mapstructure:"userbot" json:"userbot"`
}

type userbotConfig struct {
	Enable  bool   `toml:"enable" mapstructure:"enable"`
	Session string `toml:"session" mapstructure:"session"`
}

type proxyConfig struct {
	Enable bool   `toml:"enable" mapstructure:"enable"`
	URL    string `toml:"url" mapstructure:"url"`
}

var Cfg *Config

func (c Config) GetStorageByName(name string) storage.StorageConfig {
	for _, storage := range c.Storages {
		if storage.GetName() == name {
			return storage
		}
	}
	return nil
}

func Init(ctx context.Context) error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/saveany/")
	viper.SetConfigType("toml")
	viper.SetEnvPrefix("SAVEANY")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetDefault("lang", "zh-Hans")

	viper.SetDefault("workers", 3)
	viper.SetDefault("retry", 3)
	viper.SetDefault("threads", 4)

	viper.SetDefault("telegram.app_id", 1025907)
	viper.SetDefault("telegram.app_hash", "452b0359b988148995f22ff0f4229750")
	viper.SetDefault("telegram.timeout", 60)
	viper.SetDefault("telegram.flood_retry", 5)
	viper.SetDefault("telegram.rpc_retry", 5)
	viper.SetDefault("telegram.userbot.enable", false)
	viper.SetDefault("telegram.userbot.session", "data/usersession.db")

	viper.SetDefault("temp.base_path", "cache/")
	viper.SetDefault("temp.cache_ttl", 30)

	viper.SetDefault("log.level", "INFO")

	viper.SetDefault("db.path", "data/saveany.db")
	viper.SetDefault("db.session", "data/session.db")
	viper.SetDefault("db.expire", 86400*5)

	if err := viper.SafeWriteConfigAs("config.toml"); err != nil {
		if _, ok := err.(viper.ConfigFileAlreadyExistsError); !ok {
			return fmt.Errorf("error saving default config: %w", err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file, ", err)
		os.Exit(1)
	}

	Cfg = &Config{}

	if err := viper.Unmarshal(Cfg); err != nil {
		fmt.Println("Error unmarshalling config file, ", err)
		os.Exit(1)
	}

	storagesConfig, err := storage.LoadStorageConfigs(viper.GetViper())
	if err != nil {
		return fmt.Errorf("error loading storage configs: %w", err)
	}
	Cfg.Storages = storagesConfig

	storageNames := make(map[string]struct{})
	for _, storage := range Cfg.Storages {
		if _, ok := storageNames[storage.GetName()]; ok {
			return errors.New(i18n.TWithoutInit(Cfg.Lang, i18nk.ConfigInvalidDuplicateStorageName, map[string]any{
				"Name": storage.GetName(),
			}))
		}
		storageNames[storage.GetName()] = struct{}{}
	}

	fmt.Println(i18n.TWithoutInit(Cfg.Lang, i18nk.LoadedStorages, map[string]any{
		"Count": len(Cfg.Storages),
	}))
	for _, storage := range Cfg.Storages {
		fmt.Printf("  - %s (%s)\n", storage.GetName(), storage.GetType())
	}

	if Cfg.Workers < 1 || Cfg.Retry < 1 {
		return errors.New(i18n.TWithoutInit(Cfg.Lang, i18nk.ConfigInvalidWorkersOrRetry, map[string]any{
			"Workers": Cfg.Workers,
			"Retry":   Cfg.Retry,
		}))
	}

	for _, storage := range Cfg.Storages {
		storages = append(storages, storage.GetName())
	}
	for _, user := range Cfg.Users {
		userIDs = append(userIDs, user.ID)
		if user.Blacklist {
			userStorages[user.ID] = slice.Compact(slice.Difference(storages, user.Storages))
		} else {
			userStorages[user.ID] = user.Storages
		}
	}

	return nil
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
