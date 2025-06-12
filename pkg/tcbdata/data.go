package tcbdata

import (
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type Add struct {
	File        tfile.TGFile
	StorageName string
	DirID       int64
}

type SetDefaultStorage struct {
	StorageName string
}
