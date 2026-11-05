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
	Lang         string `toml:"lang" mapstructure:"lang" json:"lang"`
	Workers      int    `toml:"workers" mapstructure:"workers" json:"workers"`
	Retry        int    `toml:"retry" mapstructure:"retry" json:"retry"`
	NoCleanCache bool   `toml:"no_clean_cache" mapstructure:"no_clean_cache" json:"no_clean_cache"`
	Threads      int    `toml:"threads" mapstructure:"threads" json:"threads"`
	// DownloadThreads overrides automatic per-file chunk concurrency when > 0.
	DownloadThreads int `toml:"download_threads" mapstructure:"download_threads" json:"download_threads"`
	// MaxDownloadThreads caps automatic per-file chunk concurrency.
	MaxDownloadThreads int `toml:"max_download_threads" mapstructure:"max_download_threads" json:"max_download_threads"`
	// TransferBudget caps concurrent Telegram part RPCs; 0 derives workers * threads.
	TransferBudget int `toml:"transfer_budget" mapstructure:"transfer_budget" json:"transfer_budget"`
	// RelayMinSizeMB controls the smallest file eligible for helper downloads.
	RelayMinSizeMB int `toml:"relay_min_size_mb" mapstructure:"relay_min_size_mb" json:"relay_min_size_mb"`
	// ChunkRetries is the retry count for transient errors on one file chunk.
	ChunkRetries int                     `toml:"chunk_retries" mapstructure:"chunk_retries" json:"chunk_retries"`
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
	Metadata     metadataConfig          `toml:"metadata" mapstructure:"metadata" json:"metadata"`
	Media        mediaConfig             `toml:"media" mapstructure:"media" json:"media"`
}

type metadataConfig struct {
	Enabled     bool `toml:"enabled" mapstructure:"enabled" json:"enabled"`
	IncludeText bool `toml:"include_text" mapstructure:"include_text" json:"include_text"`
	Sidecar     bool `toml:"sidecar" mapstructure:"sidecar" json:"sidecar"`
}

type mediaConfig struct {
	// AllowedMediaTypes is empty when all supported media types are allowed.
	AllowedMediaTypes []string `toml:"allowed_media_types" mapstructure:"allowed_media_types" json:"allowed_media_types"`
}

type webConfig struct {
	Enable       bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	Listen       string `toml:"listen" mapstructure:"listen" json:"listen"`
	APIToken     string `toml:"api_token" mapstructure:"api_token" json:"api_token"`
	SecureCookie bool   `toml:"secure_cookie" mapstructure:"secure_cookie" json:"secure_cookie"`
}

const (
	defaultBotSessionFile     = "bot_session.db"
	defaultUserbotSessionFile = "userbot_session.db"
	defaultWebListen          = "127.0.0.1:28080"
)

var current atomic.Pointer[Config]

// DefaultBotSessionPath returns the v1 bot session database path.
func DefaultBotSessionPath(dataDir string) string {
	return filepath.Join(dataDir, defaultBotSessionFile)
}

// DefaultUserbotSessionPath returns the v1 userbot session database path.
func DefaultUserbotSessionPath(dataDir string) string {
	return filepath.Join(dataDir, defaultUserbotSessionFile)
}

func init() {
	cfg := Default("./data")
	current.Store(&cfg)
}

// C returns a snapshot of the current configuration. Callers must not mutate it.
func C() Config {
	return *current.Load()
}

// TransferBudget is the upper bound for concurrent Telegram file-part RPCs
// (download and upload combined). It keeps the per-file thread setting from
// multiplying with task and element concurrency.
func TransferBudget() int {
	cfg := current.Load()
	if cfg.TransferBudget > 0 {
		return cfg.TransferBudget
	}
	return max(cfg.Workers, 1) * max(cfg.Threads, 1)
}

// Set replaces the current configuration snapshot.
func Set(cfg Config) {
	current.Store(&cfg)
}

func DefaultsMap(dataDir string) map[string]any {
	return map[string]any{
		"lang":                 "zh-Hans",
		"workers":              3,
		"retry":                3,
		"no_clean_cache":       false,
		"threads":              4,
		"download_threads":     0,
		"max_download_threads": 16,
		"transfer_budget":      0,
		"relay_min_size_mb":    8,
		"chunk_retries":        20,
		"log.level":            "debug",

		"cache.ttl":          86400,
		"cache.num_counters": 1e5,
		"cache.max_cost":     1e6,

		"telegram.app_id":              1025907,
		"telegram.app_hash":            "452b0359b988148995f22ff0f4229750",
		"telegram.rpc_retry":           5,
		"telegram.userbot.enable":      false,
		"telegram.userbot.session":     DefaultUserbotSessionPath(dataDir),
		"telegram.relay_chat_id":       0,
		"telegram.relay_keep_messages": true,

		"temp.base_path": "cache/",

		"db.session": DefaultBotSessionPath(dataDir),

		"web.enable":        true,
		"web.listen":        defaultWebListen,
		"web.api_token":     "",
		"web.secure_cookie": false,

		"ytdlp.recode":             "mp4",
		"ytdlp.filename_template":  DefaultYtdlpFilenameTemplate,
		"ytdlp.max_height":         0,
		"ytdlp.format":             "",
		"ytdlp.restrict_filenames": false,

		"metadata.enabled":          false,
		"metadata.include_text":     true,
		"metadata.sidecar":          false,
		"media.allowed_media_types": []string{},
	}
}

func Default(dataDir string) Config {
	return Config{
		Lang:               "zh-Hans",
		Workers:            3,
		Retry:              3,
		Threads:            4,
		DownloadThreads:    0,
		MaxDownloadThreads: 16,
		TransferBudget:     0,
		RelayMinSizeMB:     8,
		ChunkRetries:       20,
		Log:                logConfig{Level: "debug"},
		Cache: cacheConfig{
			TTL:         86400,
			NumCounters: 1e5,
			MaxCost:     1e6,
		},
		Temp: tempConfig{BasePath: "cache/"},
		DB:   dbConfig{Session: DefaultBotSessionPath(dataDir)},
		Telegram: telegramConfig{
			AppID:             1025907,
			AppHash:           "452b0359b988148995f22ff0f4229750",
			RpcRetry:          5,
			Userbot:           userbotConfig{Session: DefaultUserbotSessionPath(dataDir)},
			RelayKeepMessages: true,
		},
		Ytdlp: YtdlpConfig{
			Recode:           "mp4",
			FilenameTemplate: DefaultYtdlpFilenameTemplate,
		},
		Web: webConfig{
			Enable: true,
			Listen: defaultWebListen,
		},
		Metadata: metadataConfig{IncludeText: true},
		Media:    mediaConfig{},
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

func (c Config) GetStorageIDByName(name string) uint {
	if st := c.GetStorageByName(name); st != nil {
		return st.GetID()
	}
	return 0
}

func (c Config) GetStorageByID(id uint) storage.StorageConfig {
	if id == 0 {
		return nil
	}
	for _, st := range c.Storages {
		if st.GetID() == id {
			return st
		}
	}
	return nil
}

func (c Config) GetStorageNameByID(id uint) string {
	if id == 0 {
		return ""
	}
	if st := c.GetStorageByID(id); st != nil {
		return st.GetName()
	}
	return ""
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
	return randomBase64(15)
}

// GenerateToken returns a URL-safe token with 256 bits of entropy.
func GenerateToken() string {
	return randomBase64(32)
}

func randomBase64(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("failed to read random bytes: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
