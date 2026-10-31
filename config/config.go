package config

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/krau/SaveAny-Bot/config/storage"
	"golang.org/x/net/proxy"
)

type Config struct {
	Lang         string                  `toml:"lang" mapstructure:"lang" json:"lang"`
	Workers      int                     `toml:"workers" mapstructure:"workers" json:"workers"`
	Retry        int                     `toml:"retry" mapstructure:"retry" json:"retry"`
	NoCleanCache bool                    `toml:"no_clean_cache" mapstructure:"no_clean_cache" json:"no_clean_cache"`
	Threads      int                     `toml:"threads" mapstructure:"threads" json:"threads"`
	Stream       bool                    `toml:"stream" mapstructure:"stream" json:"stream"`
	Proxy        string                  `toml:"proxy" mapstructure:"proxy" json:"proxy"`
	Log          logConfig               `toml:"log" mapstructure:"log" json:"log"`
	Aria2        aria2Config             `toml:"aria2" mapstructure:"aria2" json:"aria2"`
	Cache        cacheConfig             `toml:"cache" mapstructure:"cache" json:"cache"`
	Users        []UserConfig            `toml:"users" mapstructure:"users" json:"-"`
	Temp         tempConfig              `toml:"temp" mapstructure:"temp" json:"temp"`
	DB           dbConfig                `toml:"db" mapstructure:"db" json:"db"`
	Telegram     telegramConfig          `toml:"telegram" mapstructure:"telegram" json:"telegram"`
	Storages     []storage.StorageConfig `toml:"-" mapstructure:"-" json:"-"`
	Parser       parserConfig            `toml:"parser" mapstructure:"parser" json:"parser"`
	Hook         hookConfig              `toml:"hook" mapstructure:"hook" json:"hook"`
	Ytdlp        YtdlpConfig             `toml:"ytdlp" mapstructure:"ytdlp" json:"ytdlp"`
	Web          webConfig               `toml:"web" mapstructure:"web" json:"web"`
}

type webConfig struct {
	Enable       bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	Listen       string `toml:"listen" mapstructure:"listen" json:"listen"`
	APIToken     string `toml:"api_token" mapstructure:"api_token" json:"api_token"`
	SecureCookie bool   `toml:"secure_cookie" mapstructure:"secure_cookie" json:"secure_cookie"`
}

var current atomic.Pointer[Config]

func init() {
	cfg := Default("./data")
	current.Store(&cfg)
}

// C returns a snapshot of the current configuration. Callers must not mutate it.
func C() Config {
	return *current.Load()
}

// Set replaces the current configuration snapshot.
func Set(cfg Config) {
	current.Store(&cfg)
}

func DefaultsMap(dataDir string) map[string]any {
	return map[string]any{
		"lang":           "zh-Hans",
		"workers":        3,
		"retry":          3,
		"no_clean_cache": false,
		"threads":        4,
		"log.level":      "debug",

		"cache.ttl":          86400,
		"cache.num_counters": 1e5,
		"cache.max_cost":     1e6,

		"telegram.app_id":          1025907,
		"telegram.app_hash":        "452b0359b988148995f22ff0f4229750",
		"telegram.rpc_retry":       5,
		"telegram.userbot.enable":  false,
		"telegram.userbot.session": filepath.Join(dataDir, "usersession.db"),

		"temp.base_path": "cache/",

		"db.session": filepath.Join(dataDir, "session.db"),

		"web.enable":        true,
		"web.listen":        "0.0.0.0:8080",
		"web.api_token":     "",
		"web.secure_cookie": false,

		"ytdlp.recode":             "mp4",
		"ytdlp.filename_template":  DefaultYtdlpFilenameTemplate,
		"ytdlp.max_height":         0,
		"ytdlp.format":             "",
		"ytdlp.restrict_filenames": false,
	}
}

func Default(dataDir string) Config {
	return Config{
		Lang:    "zh-Hans",
		Workers: 3,
		Retry:   3,
		Threads: 4,
		Log:     logConfig{Level: "debug"},
		Cache: cacheConfig{
			TTL:         86400,
			NumCounters: 1e5,
			MaxCost:     1e6,
		},
		Temp: tempConfig{BasePath: "cache/"},
		DB:   dbConfig{Session: filepath.Join(dataDir, "session.db")},
		Telegram: telegramConfig{
			AppID:    1025907,
			AppHash:  "452b0359b988148995f22ff0f4229750",
			RpcRetry: 5,
			Userbot:  userbotConfig{Session: filepath.Join(dataDir, "usersession.db")},
		},
		Ytdlp: YtdlpConfig{
			Recode:           "mp4",
			FilenameTemplate: DefaultYtdlpFilenameTemplate,
		},
		Web: webConfig{
			Enable: true,
			Listen: "0.0.0.0:8080",
		},
	}
}

func (c Config) GetStorageByName(name string) storage.StorageConfig {
	for _, st := range c.Storages {
		if st.GetName() == name {
			return st
		}
	}
	return nil
}

// ProxyTransport builds the HTTP transport used for the configured proxy.
func ProxyTransport(proxyStr string) (*http.Transport, error) {
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

// GeneratePassword returns a URL-safe random password with ~120 bits of entropy.
func GeneratePassword() string {
	buf := make([]byte, 15)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("failed to read random bytes: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
