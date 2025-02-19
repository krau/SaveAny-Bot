package config

import (
	"github.com/duke-git/lancet/v2/slice"
)

type userConfig struct {
	ID        int64    `toml:"id" mapstructure:"id" json:"id"`                      // telegram user id
	Storages  []string `toml:"storages" mapstructure:"storages" json:"storages"`    // storage names
	Blacklist bool     `toml:"blacklist" mapstructure:"blacklist" json:"blacklist"` // 黑名单模式, storage names 中的存储将不会被使用, 默认为白名单模式
}

func (c *Config) GetStorageNamesByUserID(userID int64) []string {
	for _, user := range c.Users {
		if user.ID == userID {
			if user.Blacklist {
				allStorages := make([]string, 0, len(c.Storages))
				for _, storage := range c.Storages {
					allStorages = append(allStorages, storage.GetName())
				}
				return slice.Compact(slice.Difference(allStorages, user.Storages))
			} else {
				return user.Storages
			}
		}
	}
	return nil
}
