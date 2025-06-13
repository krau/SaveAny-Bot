package tcbdata

import (
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const (
	TypeAddOne     = "addone"
	TypeAddBatch   = "addbatch"
	TypeSetDefault = "setdefault"
)

type Add struct {
	File        tfile.TGFile
	StorageName string
	DirID       int64
}

type AddBatch struct {
	Files           []tfile.TGFile
	SelectedStorage string
}

type SetDefaultStorage struct {
	StorageName string
}
