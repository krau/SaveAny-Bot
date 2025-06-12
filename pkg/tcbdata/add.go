package tcbdata

import "github.com/krau/SaveAny-Bot/pkg/tfile"

type Add struct {
	File        tfile.TGFile
	StorageName string
	DirID       int64
	ChatID      int64 // Where the message was sent to bot
}

type SetDefaultStorage struct {
	StorageName string
}
