package tcbdata

import (
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const (
	TypeAdd        = "add"
	TypeSetDefault = "setdefault"
)

type Add struct {
	Files            []tfile.TGFile
	AsBatch          bool
	SelectedStorName string
	DirID            uint
	SettedDir        bool
}

type SetDefaultStorage struct {
	StorageName string
}
