package config

import (
	"strconv"

	"github.com/krau/SaveAny-Bot/types"
	"gorm.io/datatypes"
)

// for compatibility
type deprecatedStorageConfig struct {
	Alist  alistConfig  `toml:"alist" mapstructure:"alist"`
	Local  localConfig  `toml:"local" mapstructure:"local"`
	Webdav webdavConfig `toml:"webdav" mapstructure:"webdav"`
}

type alistConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	URL      string `toml:"url" mapstructure:"url" json:"url"`
	Username string `toml:"username" mapstructure:"username" json:"username"`
	Password string `toml:"password" mapstructure:"password" json:"password"`
	Token    string `toml:"token" mapstructure:"token" json:"token"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	TokenExp int64  `toml:"token_exp" mapstructure:"token_exp" json:"token_exp"`
}

func (a *alistConfig) ToJSON() datatypes.JSON {
	tokenExp := strconv.FormatInt(a.TokenExp, 10)
	return datatypes.JSON([]byte(`{"url":"` + a.URL + `","username":"` + a.Username + `","password":"` + a.Password + `","token":"` + a.Token + `","base_path":"` + a.BasePath + `","token_exp":` + tokenExp + `}`))
}

type localConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}

func (l *localConfig) ToJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{"base_path":"` + l.BasePath + `"}`))
}

type webdavConfig struct {
	Enable   bool   `toml:"enable" mapstructure:"enable" json:"enable"`
	URL      string `toml:"url" mapstructure:"url" json:"url"`
	Username string `toml:"username" mapstructure:"username" json:"username"`
	Password string `toml:"password" mapstructure:"password" json:"password"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}

func (w *webdavConfig) ToJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{"url":"` + w.URL + `","username":"` + w.Username + `","password":"` + w.Password + `","base_path":"` + w.BasePath + `"}`))
}

func transformDeprecatedStorageConfig() {
	if Cfg.DeprecatedStorage.Alist.Enable {
		alistStorage := &AlistStorageConfig{
			NewStorageConfig: NewStorageConfig{
				Name:   "Alist",
				Enable: true,
				Type:   string(types.StorageTypeAlist),
			},
			URL:      Cfg.DeprecatedStorage.Alist.URL,
			Username: Cfg.DeprecatedStorage.Alist.Username,
			Password: Cfg.DeprecatedStorage.Alist.Password,
			Token:    Cfg.DeprecatedStorage.Alist.Token,
			BasePath: Cfg.DeprecatedStorage.Alist.BasePath,
			TokenExp: Cfg.DeprecatedStorage.Alist.TokenExp,
		}
		Cfg.Storages = append(Cfg.Storages, alistStorage)
	}
	if Cfg.DeprecatedStorage.Local.Enable {
		localStorage := &LocalStorageConfig{
			NewStorageConfig: NewStorageConfig{
				Name:   "Local",
				Enable: true,
				Type:   string(types.StorageTypeLocal),
			},
			BasePath: Cfg.DeprecatedStorage.Local.BasePath,
		}
		Cfg.Storages = append(Cfg.Storages, localStorage)
	}
	if Cfg.DeprecatedStorage.Webdav.Enable {
		webdavStorage := &WebdavStorageConfig{
			NewStorageConfig: NewStorageConfig{
				Name:   "Webdav",
				Enable: true,
				Type:   string(types.StorageTypeWebdav),
			},
			URL:      Cfg.DeprecatedStorage.Webdav.URL,
			Username: Cfg.DeprecatedStorage.Webdav.Username,
			Password: Cfg.DeprecatedStorage.Webdav.Password,
			BasePath: Cfg.DeprecatedStorage.Webdav.BasePath,
		}
		Cfg.Storages = append(Cfg.Storages, webdavStorage)
	}
}
