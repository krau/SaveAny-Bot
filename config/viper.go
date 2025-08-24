package config

import (
	"context"
	"errors"
	"fmt"
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

	Cache    cacheConfig             `toml:"cache" mapstructure:"cache" json:"cache"`
	Users    []userConfig            `toml:"users" mapstructure:"users" json:"users"`
	Temp     tempConfig              `toml:"temp" mapstructure:"temp"`
	DB       dbConfig                `toml:"db" mapstructure:"db"`
	Telegram telegramConfig          `toml:"telegram" mapstructure:"telegram"`
	Storages []storage.StorageConfig `toml:"-" mapstructure:"-" json:"storages"`
	Parser   parserConfig            `toml:"parser" mapstructure:"parser" json:"parser"`
	Hook     hookConfig              `toml:"hook" mapstructure:"hook" json:"hook"`
}

var cfg = &Config{}

func C() Config {
	return *cfg
}

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

	defaultConfigs := map[string]any{
		// 基础配置
		"lang":    "zh-Hans",
		"workers": 3,
		"retry":   3,
		"threads": 4,

		// 缓存配置
		"cache.ttl":          86400,
		"cache.num_counters": 1e5,
		"cache.max_cost":     1e6,

		// Telegram
		"telegram.app_id":          1025907,
		"telegram.app_hash":        "452b0359b988148995f22ff0f4229750",
		"telegram.rpc_retry":       5,
		"telegram.userbot.enable":  false,
		"telegram.userbot.session": "data/usersession.db",

		// 临时目录
		"temp.base_path": "cache/",

		// 数据库
		"db.path":    "data/saveany.db",
		"db.session": "data/session.db",
	}

	for key, value := range defaultConfigs {
		viper.SetDefault(key, value)
	}

	if err := viper.SafeWriteConfigAs("config.toml"); err != nil {
		if _, ok := err.(viper.ConfigFileAlreadyExistsError); !ok {
			return fmt.Errorf("error saving default config: %w", err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file, ", err)
		return err
	}

	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Println("Error unmarshalling config file, ", err)
		return err
	}

	storagesConfig, err := storage.LoadStorageConfigs(viper.GetViper())
	if err != nil {
		return fmt.Errorf("error loading storage configs: %w", err)
	}
	cfg.Storages = storagesConfig

	storageNames := make(map[string]struct{})
	for _, storage := range cfg.Storages {
		if _, ok := storageNames[storage.GetName()]; ok {
			return errors.New(i18n.TWithoutInit(cfg.Lang, i18nk.ConfigInvalidDuplicateStorageName, map[string]any{
				"Name": storage.GetName(),
			}))
		}
		storageNames[storage.GetName()] = struct{}{}
	}

	fmt.Println(i18n.TWithoutInit(cfg.Lang, i18nk.LoadedStorages, map[string]any{
		"Count": len(cfg.Storages),
	}))
	for _, storage := range cfg.Storages {
		fmt.Printf("  - %s (%s)\n", storage.GetName(), storage.GetType())
	}

	if cfg.Workers < 1 || cfg.Retry < 1 {
		return errors.New(i18n.TWithoutInit(cfg.Lang, i18nk.ConfigInvalidWorkersOrRetry, map[string]any{
			"Workers": cfg.Workers,
			"Retry":   cfg.Retry,
		}))
	}

	for _, storage := range cfg.Storages {
		storages = append(storages, storage.GetName())
	}
	for _, user := range cfg.Users {
		userIDs = append(userIDs, user.ID)
		if user.Blacklist {
			userStorages[user.ID] = slice.Compact(slice.Difference(storages, user.Storages))
		} else {
			userStorages[user.ID] = user.Storages
		}
	}
	return nil
}
