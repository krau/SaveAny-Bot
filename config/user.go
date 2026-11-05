package config

import (
	"github.com/duke-git/lancet/v2/slice"

	"github.com/krau/SaveAny-Bot/pkg/enums/storagefilter"
)

type UserConfig struct {
	ID                 int64              `toml:"id" mapstructure:"id" json:"id"` // telegram user id
	Enabled            bool               `toml:"enabled" mapstructure:"enabled" json:"enabled"`
	DefaultStorageID   uint               `toml:"default_storage_id" mapstructure:"default_storage_id" json:"default_storage_id"`
	StorageFilterIDs   []uint             `toml:"storage_filter_ids" mapstructure:"storage_filter_ids" json:"storage_filter_ids"`
	StorageFilterNames []string           `toml:"storages" mapstructure:"storages" json:"-"` // legacy config import only
	StorageFilterMode  storagefilter.Mode `toml:"storage_filter_mode" mapstructure:"storage_filter_mode" json:"storage_filter_mode"`
	// LegacyBlacklist is only used while importing a v0 config file. New
	// database-backed settings always use StorageFilterMode.
	LegacyBlacklist bool `toml:"blacklist" mapstructure:"blacklist" json:"-"`
}

// FilterMode resolves the storage filter mode, including the v0 blacklist
// flag used during config import.
func (u UserConfig) FilterMode() storagefilter.Mode {
	if u.StorageFilterMode.Valid() {
		return u.StorageFilterMode
	}
	if u.LegacyBlacklist {
		return storagefilter.Blacklist
	}
	return storagefilter.Whitelist
}

func (u UserConfig) filterIDs(c Config) []uint {
	if len(u.StorageFilterIDs) > 0 || len(u.StorageFilterNames) == 0 {
		return u.StorageFilterIDs
	}
	ids := make([]uint, 0, len(u.StorageFilterNames))
	for _, name := range u.StorageFilterNames {
		if id := c.GetStorageIDByName(name); id != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetStorageIDsByUserID returns the storage IDs accessible to a user.
func (c Config) GetStorageIDsByUserID(userID int64) []uint {
	for _, u := range c.Users {
		if u.ID != userID || !u.Enabled {
			continue
		}
		selected := u.filterIDs(c)
		if u.FilterMode() == storagefilter.Whitelist {
			return selected
		}
		selectedSet := make(map[uint]struct{}, len(selected))
		for _, id := range selected {
			selectedSet[id] = struct{}{}
		}
		ids := make([]uint, 0, len(c.Storages))
		for _, st := range c.Storages {
			if _, denied := selectedSet[st.GetID()]; !denied {
				ids = append(ids, st.GetID())
			}
		}
		return ids
	}
	return nil
}

func (c Config) GetStorageNamesByUserID(userID int64) []string {
	ids := c.GetStorageIDsByUserID(userID)
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name := c.GetStorageNameByID(id); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (c Config) GetUsersID() []int64 {
	ids := make([]int64, 0, len(c.Users))
	for _, u := range c.Users {
		if u.Enabled {
			ids = append(ids, u.ID)
		}
	}
	return ids
}

func (c Config) HasStorageID(userID int64, storageID uint) bool {
	return slice.Contain(c.GetStorageIDsByUserID(userID), storageID)
}

func (c Config) HasStorage(userID int64, storageName string) bool {
	return slice.Contain(c.GetStorageNamesByUserID(userID), storageName)
}
