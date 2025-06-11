package tfile

import (
	"github.com/gotd/td/tg"
)

type TGFile interface {
	Location() tg.InputFileLocationClass
	Size() int64
	Name() string
}

type tgFile struct {
	location tg.InputFileLocationClass
	size     int64
	name     string
}

func (f *tgFile) Location() tg.InputFileLocationClass {
	return f.location
}
func (f *tgFile) Size() int64 {
	return f.size
}
func (f *tgFile) Name() string {
	return f.name
}

func NewTGFile(location tg.InputFileLocationClass, size int64, name string) TGFile {
	return &tgFile{
		location: location,
		size:     size,
		name:     name,
	}
}
