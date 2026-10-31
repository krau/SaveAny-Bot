package config

import (
	"github.com/duke-git/lancet/v2/slice"
)

type UserConfig struct {
	ID        int64    `toml:"id" mapstructure:"id" json:"id"`                      // telegram user id
	Enabled   bool     `toml:"enabled" mapstructure:"enabled" json:"enabled"`       // whether the user may use the bot
	Storages  []string `toml:"storages" mapstructure:"storages" json:"storages"`    // storage names
	Blacklist bool     `toml:"blacklist" mapstructure:"blacklist" json:"blacklist"` // blacklist mode: listed storages are denied
}

func (c Config) GetStorageNamesByUserID(userID int64) []string {
	for _, u := range c.Users {
		if u.ID != userID || !u.Enabled {
			continue
		}
		if !u.Blacklist {
			return u.Storages
		}
		all := make([]string, 0, len(c.Storages))
		for _, st := range c.Storages {
			all = append(all, st.GetName())
		}
		return slice.Compact(slice.Difference(all, u.Storages))
	}
	return nil
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

func (c Config) HasStorage(userID int64, storageName string) bool {
	return slice.Contain(c.GetStorageNamesByUserID(userID), storageName)
}
