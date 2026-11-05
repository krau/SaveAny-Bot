package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/krau/SaveAny-Bot/config"
	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/database"
)

const (
	legacyImportedKey        = "meta.legacy_config_imported"
	legacyBotSessionFile     = "session.db"
	legacyUserbotSessionFile = "usersession.db"
)

func LegacyImported(ctx context.Context) bool {
	_, ok, err := database.GetSetting(ctx, legacyImportedKey)
	if err != nil {
		log.Errorf("Failed to read legacy import marker: %v", err)
		return false
	}
	return ok
}

// LegacyImportTime returns the time at which a v0 configuration was imported.
// The boolean reports whether an import marker exists, even if its value is
// malformed.
func LegacyImportTime(ctx context.Context) (time.Time, bool) {
	value, ok, err := database.GetSetting(ctx, legacyImportedKey)
	if err != nil {
		log.Errorf("Failed to read legacy import marker: %v", err)
		return time.Time{}, false
	}
	if !ok {
		return time.Time{}, false
	}
	importedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, true
	}
	return importedAt, true
}

// ImportLegacy imports a legacy TOML configuration exactly once.
func ImportLegacy(ctx context.Context, src, dataDir string) error {
	data, err := readLegacyConfig(src)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	v := viper.New()
	v.SetConfigType("toml")
	for key, value := range config.DefaultsMap(dataDir) {
		v.SetDefault(key, value)
	}
	if err := v.ReadConfig(strings.NewReader(string(data))); err != nil {
		return fmt.Errorf("failed to parse legacy config: %w", err)
	}
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal legacy config: %w", err)
	}
	legacyBotSession := strings.TrimSpace(v.GetString("db.session"))
	legacyUserbotSession := strings.TrimSpace(v.GetString("telegram.userbot.session"))
	cfg.DB.Session = config.DefaultBotSessionPath(dataDir)
	cfg.Telegram.Userbot.Session = config.DefaultUserbotSessionPath(dataDir)
	if v.InConfig("db.session") && legacyBotSession != "" {
		log.Warn("Ignoring legacy Bot session database; v1 uses a new session format", "path", legacyBotSession)
	}
	if v.InConfig("telegram.userbot.session") && legacyUserbotSession != "" {
		log.Warn("Ignoring legacy Userbot session database; v1 uses a new session format", "path", legacyUserbotSession)
	}
	baseConfigs, err := storcfg.LoadAllBaseConfigs(v)
	if err != nil {
		return fmt.Errorf("failed to load legacy storages: %w", err)
	}

	existing, err := database.ListSettings(ctx)
	if err != nil {
		return err
	}
	groups := make(map[string]string)
	for _, group := range []string{GroupCore, GroupLog, GroupDB, GroupTelegram, GroupAria2, GroupCache, GroupTemp, GroupYtdlp, GroupParser, GroupHook} {
		if _, ok := existing[group]; ok {
			continue
		}
		value, err := groupValue(cfg, group)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		groups[group] = string(raw)
	}
	if _, ok := existing[GroupWeb]; !ok {
		listen := cfgListenFromAPI(v)
		web := map[string]any{
			"enable":        true,
			"listen":        listen,
			"api_token":     v.GetString("api.token"),
			"secure_cookie": false,
		}
		raw, err := json.Marshal(web)
		if err != nil {
			return fmt.Errorf("encode legacy web settings: %w", err)
		}
		groups[GroupWeb] = string(raw)
	}

	legacyChatIDs := make([]int64, 0, len(cfg.Users))
	for _, u := range cfg.Users {
		legacyChatIDs = append(legacyChatIDs, u.ID)
	}

	err = database.Transaction(ctx, func(tx *gorm.DB) error {
		for key, value := range groups {
			if err := tx.Save(&database.Setting{Key: key, Value: value}).Error; err != nil {
				return err
			}
		}
		for _, base := range baseConfigs {
			var count int64
			if err := tx.Model(&database.StorageRecord{}).Where("name = ?", base.Name).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			raw, err := json.Marshal(base.RawConfig)
			if err != nil {
				return err
			}
			if err := tx.Create(&database.StorageRecord{
				Name: base.Name, Type: base.Type, Enable: base.Enable, ConfigJSON: string(raw),
			}).Error; err != nil {
				return err
			}
		}
		storageIDs := make(map[string]uint, len(baseConfigs))
		for _, base := range baseConfigs {
			var record database.StorageRecord
			if err := tx.Where("name = ?", base.Name).First(&record).Error; err != nil {
				return err
			}
			storageIDs[base.Name] = record.ID
		}
		for _, u := range cfg.Users {
			filterIDs := make([]uint, 0, len(u.StorageFilterIDs)+len(u.StorageFilterNames))
			filterIDs = append(filterIDs, u.StorageFilterIDs...)
			if len(filterIDs) == 0 {
				for _, name := range u.StorageFilterNames {
					if id := storageIDs[name]; id != 0 {
						filterIDs = append(filterIDs, id)
					}
				}
			}
			raw, err := json.Marshal(filterIDs)
			if err != nil {
				return err
			}
			mode := u.FilterMode()
			var user database.User
			findErr := tx.Where("chat_id = ?", u.ID).First(&user).Error
			if findErr != nil && !database.IsNotFound(findErr) {
				return findErr
			}
			if database.IsNotFound(findErr) {
				if err := tx.Create(&database.User{
					ChatID: u.ID, Enabled: true, StorageFilterIDsJSON: string(raw), StorageFilterMode: mode,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&user).Updates(map[string]any{
				"enabled": true, "storage_filter_ids": string(raw), "storage_filter_mode": string(mode),
			}).Error; err != nil {
				return err
			}
		}
		query := tx.Model(&database.User{})
		if len(legacyChatIDs) > 0 {
			query = query.Where("chat_id NOT IN ?", legacyChatIDs)
		}
		if err := query.Update("enabled", false).Error; err != nil {
			return err
		}
		return tx.Save(&database.Setting{
			Key: legacyImportedKey, Value: time.Now().UTC().Format(time.RFC3339),
		}).Error
	})
	if err != nil {
		return err
	}
	log.Info("Legacy configuration imported into the database")
	return nil
}

func cfgListenFromAPI(v *viper.Viper) string {
	host := strings.TrimSpace(v.GetString("api.host"))
	if host == "" {
		host = "0.0.0.0"
	}
	port := v.GetInt("api.port")
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func readLegacyConfig(src string) ([]byte, error) {
	if src == "" {
		for _, candidate := range []string{"./config.toml", "/etc/saveany/config.toml"} {
			if data, err := os.ReadFile(candidate); err == nil {
				log.Warn("Importing legacy configuration; config files are deprecated", "path", candidate)
				return data, nil
			}
		}
		return nil, nil
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(src)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch remote config: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch remote config: status code %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	return data, nil
}
