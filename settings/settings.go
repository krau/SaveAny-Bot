package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/config"
	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/database"
	storages "github.com/krau/SaveAny-Bot/storage"
)

const (
	GroupCore     = "core"
	GroupLog      = "log"
	GroupDB       = "db"
	GroupTelegram = "telegram"
	GroupAria2    = "aria2"
	GroupCache    = "cache"
	GroupTemp     = "temp"
	GroupYtdlp    = "ytdlp"
	GroupParser   = "parser"
	GroupHook     = "hook"
	GroupWeb      = "web"
)

const maskedValue = "__MASKED__"

var ErrUnknownGroup = errors.New("unknown settings group")

var (
	ErrStorageInUse  = errors.New("storage is in use")
	ErrStorageExists = errors.New("storage already exists")
)

type coreGroup struct {
	Lang         string `json:"lang"`
	Workers      int    `json:"workers"`
	Retry        int    `json:"retry"`
	Threads      int    `json:"threads"`
	Stream       bool   `json:"stream"`
	NoCleanCache bool   `json:"no_clean_cache"`
	Proxy        string `json:"proxy"`
}

type authGroup struct {
	PasswordHash string `json:"password_hash"`
}

type StorageInput struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Enable bool           `json:"enable"`
	Config map[string]any `json:"config"`
}

func groupValue(cfg config.Config, group string) (any, error) {
	switch group {
	case GroupCore:
		return coreGroup{
			Lang:         cfg.Lang,
			Workers:      cfg.Workers,
			Retry:        cfg.Retry,
			Threads:      cfg.Threads,
			Stream:       cfg.Stream,
			NoCleanCache: cfg.NoCleanCache,
			Proxy:        cfg.Proxy,
		}, nil
	case GroupLog:
		return cfg.Log, nil
	case GroupDB:
		return cfg.DB, nil
	case GroupTelegram:
		return cfg.Telegram, nil
	case GroupAria2:
		return cfg.Aria2, nil
	case GroupCache:
		return cfg.Cache, nil
	case GroupTemp:
		return cfg.Temp, nil
	case GroupYtdlp:
		return cfg.Ytdlp, nil
	case GroupParser:
		return cfg.Parser, nil
	case GroupHook:
		return cfg.Hook, nil
	case GroupWeb:
		return cfg.Web, nil
	default:
		return nil, ErrUnknownGroup
	}
}

func applyGroup(cfg *config.Config, group string, raw []byte) error {
	switch group {
	case GroupCore:
		var g coreGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return err
		}
		cfg.Lang = g.Lang
		cfg.Workers = g.Workers
		cfg.Retry = g.Retry
		cfg.Threads = g.Threads
		cfg.Stream = g.Stream
		cfg.NoCleanCache = g.NoCleanCache
		cfg.Proxy = g.Proxy
		return nil
	case GroupLog:
		return json.Unmarshal(raw, &cfg.Log)
	case GroupDB:
		return json.Unmarshal(raw, &cfg.DB)
	case GroupTelegram:
		return json.Unmarshal(raw, &cfg.Telegram)
	case GroupAria2:
		return json.Unmarshal(raw, &cfg.Aria2)
	case GroupCache:
		return json.Unmarshal(raw, &cfg.Cache)
	case GroupTemp:
		return json.Unmarshal(raw, &cfg.Temp)
	case GroupYtdlp:
		return json.Unmarshal(raw, &cfg.Ytdlp)
	case GroupParser:
		return json.Unmarshal(raw, &cfg.Parser)
	case GroupHook:
		return json.Unmarshal(raw, &cfg.Hook)
	case GroupWeb:
		return json.Unmarshal(raw, &cfg.Web)
	default:
		return ErrUnknownGroup
	}
}

// Load builds a config snapshot from the database.
func Load(ctx context.Context, dataDir string) (config.Config, error) {
	cfg := config.Default(dataDir)
	raw, err := database.ListSettings(ctx)
	if err != nil {
		return cfg, fmt.Errorf("failed to list settings: %w", err)
	}
	for group, value := range raw {
		if group == "auth" || strings.HasPrefix(group, "meta.") {
			continue
		}
		if err := applyGroup(&cfg, group, []byte(value)); err != nil {
			if errors.Is(err, ErrUnknownGroup) {
				log.Warn("Ignoring unknown settings group", "group", group)
				continue
			}
			log.Error("Invalid settings group, keeping defaults", "group", group, "error", err)
		}
	}

	users, err := database.GetAllUsers(ctx)
	if err != nil {
		return cfg, fmt.Errorf("failed to load users: %w", err)
	}
	cfg.Users = make([]config.UserConfig, 0, len(users))
	for _, u := range users {
		var ids []uint
		if u.StorageFilterIDsJSON != "" {
			if err := json.Unmarshal([]byte(u.StorageFilterIDsJSON), &ids); err != nil {
				log.Error("Invalid storage filter for user", "chat_id", u.ChatID, "error", err)
			}
		}
		cfg.Users = append(cfg.Users, config.UserConfig{
			ID:                u.ChatID,
			Enabled:           u.Enabled,
			DefaultStorageID:  u.DefaultStorageID,
			StorageFilterIDs:  ids,
			StorageFilterMode: u.StorageFilterMode.Normalize(),
		})
	}

	records, err := database.ListStorageRecords(ctx)
	if err != nil {
		return cfg, fmt.Errorf("failed to load storages: %w", err)
	}
	seen := make(map[string]struct{}, len(records))
	cfg.Storages = nil
	for _, rec := range records {
		if _, ok := seen[rec.Name]; ok {
			return cfg, fmt.Errorf("duplicate storage name: %s", rec.Name)
		}
		seen[rec.Name] = struct{}{}
		if !rec.Enable {
			continue
		}
		cfgStorage, err := storcfg.NewStorageConfigFromJSON(rec.ID, rec.Name, rec.Type, true, []byte(rec.ConfigJSON))
		if err != nil {
			log.Error("Invalid storage config, skipping", "storage", rec.Name, "error", err)
			continue
		}
		if err := cfgStorage.Validate(); err != nil {
			log.Error("Invalid storage config, skipping", "storage", rec.Name, "error", err)
			continue
		}
		cfg.Storages = append(cfg.Storages, cfgStorage)
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
	return cfg, nil
}

type Service struct {
	ctx            context.Context
	dataDir        string
	mu             sync.Mutex
	listenOverride string
}

func NewService(ctx context.Context, dataDir string) (*Service, error) {
	s := &Service{ctx: ctx, dataDir: dataDir}
	if err := s.Reload(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Config() config.Config {
	return config.C()
}

// SetListenOverride forces the in-process listen address (from --listen)
// without persisting it, so settings reloads do not lose the CLI override.
func (s *Service) SetListenOverride(listen string) {
	s.mu.Lock()
	s.listenOverride = listen
	s.mu.Unlock()
	if listen == "" {
		return
	}
	cfg := config.C()
	cfg.Web.Listen = listen
	config.Set(cfg)
}

// ReloadConfig reloads settings and users without reinitializing storages.
func (s *Service) ReloadConfig(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := Load(ctx, s.dataDir)
	if err != nil {
		return err
	}
	if s.listenOverride != "" {
		cfg.Web.Listen = s.listenOverride
	}
	config.Set(cfg)
	i18n.Init(cfg.Lang)
	if level, err := log.ParseLevel(strings.TrimSpace(cfg.Log.Level)); err == nil {
		log.SetLevel(level)
	}
	return nil
}

// Reload reloads settings, users, and storages.
func (s *Service) Reload(ctx context.Context) error {
	if err := s.ReloadConfig(ctx); err != nil {
		return err
	}
	for _, err := range storages.Reload(context.WithoutCancel(ctx)) {
		log.Errorf("Failed to initialize storage: %v", err)
	}
	return nil
}

func (s *Service) PublicSettings() (map[string]any, error) {
	cfg := s.Config()
	out := make(map[string]any)
	for _, group := range []string{GroupCore, GroupLog, GroupDB, GroupTelegram, GroupAria2, GroupCache, GroupTemp, GroupYtdlp, GroupParser, GroupHook, GroupWeb} {
		value, err := groupValue(cfg, group)
		if err != nil {
			return nil, err
		}
		out[group] = value
	}
	maskSecrets(out)
	return out, nil
}

func maskSecrets(groups map[string]any) {
	mask := func(group string, keys ...string) {
		m, ok := groups[group].(map[string]any)
		if !ok {
			raw, err := json.Marshal(groups[group])
			if err != nil {
				return
			}
			m = make(map[string]any)
			if err := json.Unmarshal(raw, &m); err != nil {
				return
			}
			groups[group] = m
		}
		for _, key := range keys {
			if v, ok := m[key].(string); ok && v != "" {
				m[key] = maskedValue
			}
		}
	}
	// Values are structs here; normalize through JSON before masking.
	for group, value := range groups {
		raw, err := json.Marshal(value)
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		groups[group] = m
	}
	mask(GroupTelegram, "token", "app_hash")
	mask(GroupAria2, "secret")
	mask(GroupWeb, "api_token")
}

func (s *Service) UpdateGroup(ctx context.Context, group string, patch json.RawMessage) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldCfg := s.Config()
	current, err := groupValue(oldCfg, group)
	if err != nil {
		return nil, err
	}
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	var currentMap map[string]any
	if err := json.Unmarshal(currentJSON, &currentMap); err != nil {
		return nil, err
	}
	var patchMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, fmt.Errorf("invalid patch: %w", err)
	}
	for key := range patchMap {
		if _, ok := currentMap[key]; !ok {
			return nil, fmt.Errorf("unknown field %q for group %s", key, group)
		}
	}
	if group == GroupWeb && s.listenOverride != "" {
		if _, provided := patchMap["listen"]; !provided {
			listen := config.Default(s.dataDir).Web.Listen
			if raw, ok, err := database.GetSetting(ctx, GroupWeb); err == nil && ok {
				var stored map[string]any
				if json.Unmarshal([]byte(raw), &stored) == nil {
					if v, ok := stored["listen"].(string); ok {
						listen = v
					}
				}
			}
			currentMap["listen"] = listen
		}
	}
	merged := mergeJSON(currentMap, patchMap)
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}

	newCfg := oldCfg
	if err := applyGroup(&newCfg, group, mergedJSON); err != nil {
		return nil, err
	}
	if err := validateGroup(&newCfg, group); err != nil {
		return nil, err
	}
	if err := database.SetSetting(ctx, group, string(mergedJSON)); err != nil {
		return nil, err
	}
	if err := s.reloadConfigLocked(ctx); err != nil {
		return nil, err
	}
	return restartRequired(oldCfg, s.Config()), nil
}

func mergeJSON(current, patch map[string]any) map[string]any {
	out := make(map[string]any, len(current))
	maps.Copy(out, current)
	for key, value := range patch {
		if s, ok := value.(string); ok && s == maskedValue {
			continue
		}
		if patchMap, ok := value.(map[string]any); ok {
			if currentMap, ok := out[key].(map[string]any); ok {
				out[key] = mergeJSON(currentMap, patchMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func validateGroup(cfg *config.Config, group string) error {
	switch group {
	case GroupCore:
		if cfg.Workers < 1 || cfg.Retry < 1 || cfg.Threads < 1 {
			return errors.New("workers, retry and threads must be at least 1")
		}
		if cfg.Proxy != "" {
			if _, err := config.ProxyTransport(cfg.Proxy); err != nil {
				return fmt.Errorf("invalid proxy: %w", err)
			}
		}
	case GroupLog:
		if _, err := log.ParseLevel(strings.TrimSpace(cfg.Log.Level)); err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}
	case GroupWeb:
		if _, _, err := net.SplitHostPort(cfg.Web.Listen); err != nil {
			return fmt.Errorf("invalid listen address: %w", err)
		}
	case GroupTelegram:
		if cfg.Telegram.AppID < 0 {
			return errors.New("app_id must be non-negative")
		}
		if id := cfg.Telegram.RelayChatID; id != 0 && (id > 0 || !strings.HasPrefix(fmt.Sprint(id), "-100")) {
			return errors.New("relay_chat_id must be 0 or a channel ID starting with -100")
		}
	}
	return nil
}

func (s *Service) reloadConfigLocked(ctx context.Context) error {
	cfg, err := Load(ctx, s.dataDir)
	if err != nil {
		return err
	}
	if s.listenOverride != "" {
		cfg.Web.Listen = s.listenOverride
	}
	config.Set(cfg)
	i18n.Init(cfg.Lang)
	if level, err := log.ParseLevel(strings.TrimSpace(cfg.Log.Level)); err == nil {
		log.SetLevel(level)
	}
	return nil
}

func restartRequired(oldCfg, newCfg config.Config) []string {
	var fields []string
	if oldCfg.Workers != newCfg.Workers {
		fields = append(fields, "core.workers")
	}
	if oldCfg.Proxy != newCfg.Proxy {
		fields = append(fields, "core.proxy")
	}
	if oldCfg.DB.Session != newCfg.DB.Session {
		fields = append(fields, "db.session")
	}
	if oldCfg.Cache != newCfg.Cache {
		fields = append(fields, "cache.ttl", "cache.num_counters", "cache.max_cost")
	}
	if oldCfg.Parser.PluginEnable != newCfg.Parser.PluginEnable || !equalStrings(oldCfg.Parser.PluginDirs, newCfg.Parser.PluginDirs) {
		fields = append(fields, "parser.plugin_enable", "parser.plugin_dirs")
	}
	if oldCfg.Hook != newCfg.Hook {
		fields = append(fields, "hook.exec")
	}
	if oldCfg.Web.Enable != newCfg.Web.Enable {
		fields = append(fields, "web.enable")
	}
	if oldCfg.Web.Listen != newCfg.Web.Listen {
		fields = append(fields, "web.listen")
	}
	return fields
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Service) EnsureAdmin(initialPassword string) error {
	if _, ok, err := database.GetSetting(s.ctx, "auth"); err != nil {
		return err
	} else if ok {
		return nil
	}
	password := initialPassword
	generated := false
	if password == "" {
		password = config.GeneratePassword()
		generated = true
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	data, err := json.Marshal(authGroup{PasswordHash: string(hash)})
	if err != nil {
		return err
	}
	if err := database.SetSetting(s.ctx, "auth", string(data)); err != nil {
		return err
	}
	if generated {
		log.Warn("Generated web admin password; change it after login", "password", password)
	}
	return nil
}

func (s *Service) VerifyAdminPassword(password string) bool {
	value, ok, err := database.GetSetting(s.ctx, "auth")
	if err != nil || !ok {
		return false
	}
	var auth authGroup
	if err := json.Unmarshal([]byte(value), &auth); err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(password)) == nil
}

func (s *Service) ChangeAdminPassword(current, next string) error {
	if !s.VerifyAdminPassword(current) {
		return errors.New("current password is incorrect")
	}
	if len(next) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), 12)
	if err != nil {
		return err
	}
	data, err := json.Marshal(authGroup{PasswordHash: string(hash)})
	if err != nil {
		return err
	}
	return database.SetSetting(s.ctx, "auth", string(data))
}

func normalizeStorageName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("storage name is required")
	}
	if utf8.RuneCountInString(name) > 64 {
		return "", errors.New("storage name must be at most 64 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errors.New("storage name must not contain control characters")
		}
	}
	return name, nil
}

func (s *Service) CreateStorage(ctx context.Context, in StorageInput) (*database.StorageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, err := normalizeStorageName(in.Name)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Type) == "" {
		return nil, errors.New("type is required")
	}
	records, err := database.ListStorageRecords(ctx)
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		if rec.Name == name {
			return nil, fmt.Errorf("%w: %s", ErrStorageExists, name)
		}
	}
	raw, err := json.Marshal(in.Config)
	if err != nil {
		return nil, err
	}
	if err := validateStorage(ctx, in.Name, in.Type, in.Enable, raw); err != nil {
		return nil, err
	}
	record := &database.StorageRecord{Name: name, Type: in.Type, Enable: in.Enable, ConfigJSON: string(raw)}
	if err := database.CreateStorageRecord(ctx, record); err != nil {
		return nil, err
	}
	if err := s.reloadAllLocked(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) UpdateStorage(ctx context.Context, id uint, in StorageInput) (*database.StorageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := database.GetStorageRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	oldName := record.Name
	name := oldName
	if strings.TrimSpace(in.Name) != "" {
		name, err = normalizeStorageName(in.Name)
		if err != nil {
			return nil, err
		}
	}
	if name != oldName {
		records, err := database.ListStorageRecords(ctx)
		if err != nil {
			return nil, err
		}
		for _, existing := range records {
			if existing.ID != record.ID && existing.Name == name {
				return nil, fmt.Errorf("%w: %s", ErrStorageExists, name)
			}
		}
	}
	var current map[string]any
	if record.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(record.ConfigJSON), &current); err != nil {
			current = make(map[string]any)
		}
	}
	if current == nil {
		current = make(map[string]any)
	}
	for key, value := range in.Config {
		if s, ok := value.(string); ok && s == maskedValue {
			continue
		}
		current[key] = value
	}
	raw, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	if err := validateStorage(ctx, name, record.Type, in.Enable, raw); err != nil {
		return nil, err
	}
	record.Name = name
	record.ConfigJSON = string(raw)
	record.Enable = in.Enable
	if err := database.UpdateStorageRecord(ctx, record); err != nil {
		return nil, err
	}
	if err := s.reloadAllLocked(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) DeleteStorage(ctx context.Context, id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := database.GetStorageRecord(ctx, id)
	if err != nil {
		return err
	}
	users, dirs, rules, err := database.StorageReferences(ctx, record.ID)
	if err != nil {
		return err
	}
	if users+dirs+rules > 0 {
		return fmt.Errorf("%w: %d user(s), %d dir(s), %d rule(s) reference %s", ErrStorageInUse, users, dirs, rules, record.Name)
	}
	if err := database.DeleteStorageRecord(ctx, id); err != nil {
		return err
	}
	return s.reloadAllLocked(ctx)
}

func (s *Service) TestStorage(ctx context.Context, id uint) error {
	record, err := database.GetStorageRecord(ctx, id)
	if err != nil {
		return err
	}
	cfg, err := storcfg.NewStorageConfigFromJSON(record.ID, record.Name, record.Type, true, []byte(record.ConfigJSON))
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	st, err := storages.NewStorage(ctx, cfg)
	if err != nil {
		return err
	}
	if closer, ok := st.(storages.StorageCloser); ok {
		defer closer.Close()
	}
	if listable, ok := st.(storages.StorageListable); ok {
		_, err = listable.ListFiles(ctx, "")
		return err
	}
	if !st.Exists(ctx, "") {
		return errors.New("storage did not accept the test request")
	}
	return nil
}

func validateStorage(ctx context.Context, name, typ string, enable bool, raw []byte) error {
	cfg, err := storcfg.NewStorageConfigFromJSON(0, name, typ, enable, raw)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	st, err := storages.NewStorage(ctx, cfg)
	if err != nil {
		return err
	}
	if closer, ok := st.(storages.StorageCloser); ok {
		closer.Close()
	}
	return nil
}

func (s *Service) reloadAllLocked(ctx context.Context) error {
	if err := s.reloadConfigLocked(ctx); err != nil {
		return err
	}
	for _, err := range storages.Reload(context.WithoutCancel(ctx)) {
		log.Errorf("Failed to initialize storage: %v", err)
	}
	return nil
}
