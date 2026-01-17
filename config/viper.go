package config

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/config/storage"
	"github.com/spf13/viper"
	"golang.org/x/net/proxy"
)

type Config struct {
	Lang         string      `toml:"lang" mapstructure:"lang" json:"lang"`
	Workers      int         `toml:"workers" mapstructure:"workers"`
	Retry        int         `toml:"retry" mapstructure:"retry"`
	NoCleanCache bool        `toml:"no_clean_cache" mapstructure:"no_clean_cache" json:"no_clean_cache"`
	Threads      int         `toml:"threads" mapstructure:"threads" json:"threads"`
	Stream       bool        `toml:"stream" mapstructure:"stream" json:"stream"`
	Proxy        string      `toml:"proxy" mapstructure:"proxy" json:"proxy"`
	Aria2        aria2Config `toml:"aria2" mapstructure:"aria2" json:"aria2"`

	Cache    cacheConfig             `toml:"cache" mapstructure:"cache" json:"cache"`
	Users    []userConfig            `toml:"users" mapstructure:"users" json:"users"`
	Temp     tempConfig              `toml:"temp" mapstructure:"temp"`
	DB       dbConfig                `toml:"db" mapstructure:"db"`
	Telegram telegramConfig          `toml:"telegram" mapstructure:"telegram"`
	Storages []storage.StorageConfig `toml:"-" mapstructure:"-" json:"storages"`
	Parser   parserConfig            `toml:"parser" mapstructure:"parser" json:"parser"`
	Hook     hookConfig              `toml:"hook" mapstructure:"hook" json:"hook"`
}

type aria2Config struct {
	Enable   bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	Url      string `toml:"url" mapstructure:"url" json:"url"`
	Secret   string `toml:"secret" mapstructure:"secret" json:"secret"`
	KeepFile bool   `toml:"keep_file" mapstructure:"keep_file" json:"keep_file"`
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

func Init(ctx context.Context, configFile ...string) error {
	viper.SetConfigType("toml")
	viper.SetEnvPrefix("SAVEANY")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// 如果指定了配置文件路径，则使用指定的配置文件
	// 配置文件支持传入一个 http(s) URL 地址
	if len(configFile) > 0 && configFile[0] != "" {
		cfg := configFile[0]
		if strings.HasPrefix(cfg, "http://") || strings.HasPrefix(cfg, "https://") {
			// 	使用远程配置文件
			resp, err := http.Get(cfg)
			if err != nil {
				return fmt.Errorf("failed to fetch remote config file: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to fetch remote config file: status code %d", resp.StatusCode)
			}
			if err := viper.ReadConfig(resp.Body); err != nil {
				return fmt.Errorf("failed to read remote config file: %w", err)
			}
		} else {
			viper.SetConfigFile(cfg)
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/saveany/")
	}

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
			return fmt.Errorf("duplicate storage name: %s", storage.GetName())
		}
		storageNames[storage.GetName()] = struct{}{}
	}

	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Threads < 1 {
		cfg.Threads = 1
	}
	if cfg.Retry < 1 {
		cfg.Retry = 1
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
	if cfg.Proxy != "" {
		http.DefaultTransport, err = newProxyTransport(cfg.Proxy)
		if err != nil {
			return fmt.Errorf("failed to create proxy transport: %w", err)
		}
	}
	return nil
}

func newProxyTransport(proxyStr string) (*http.Transport, error) {
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)

	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
		}

	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyURL.Scheme)
	}

	return transport, nil
}
