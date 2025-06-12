package tfile

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

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

func Hash(f TGFile) string {
	lb := []byte(f.Location().String())
	fb := []byte(f.Name())
	fsb := []byte(fmt.Sprintf("%d", f.Size()))

	hashBytes := append(lb, fb...)
	hashBytes = append(hashBytes, fsb...)
	hash := md5.New()
	hash.Write(hashBytes)
	return hex.EncodeToString(hash.Sum(nil))
}

type FromMediaOptions func(*tgFile)

func WithName(name string) FromMediaOptions {
	return func(f *tgFile) {
		f.name = name
	}
}

func WithNameIfEmpty(name string) FromMediaOptions {
	return func(f *tgFile) {
		if f.name == "" {
			f.name = name
		}
	}
}

func FromMedia(media tg.MessageMediaClass, opts ...FromMediaOptions) (TGFile, error) {
	switch m := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := m.Document.AsNotEmpty()
		if !ok {
			return nil, errors.New("document is empty")
		}
		fileName := ""
		for _, attribute := range document.Attributes {
			if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
				fileName = name.GetFileName()
				break
			}
		}
		file := &tgFile{
			location: document.AsInputDocumentFileLocation(),
			size:     document.Size,
			name:     fileName,
		}
		for _, opt := range opts {
			opt(file)
		}
		return file, nil
	case *tg.MessageMediaPhoto:
		photo, ok := m.Photo.AsNotEmpty()
		if !ok {
			return nil, errors.New("photo is empty")
		}
		sizes := photo.Sizes
		if len(sizes) == 0 {
			return nil, errors.New("photo sizes are empty")
		}
		photoSize := sizes[len(sizes)-1]
		size, ok := photoSize.AsNotEmpty()
		if !ok {
			return nil, errors.New("photo size is empty")
		}
		location := new(tg.InputPhotoFileLocation)
		location.ID = photo.GetID()
		location.AccessHash = photo.GetAccessHash()
		location.FileReference = photo.GetFileReference()
		location.ThumbSize = size.GetType()
		fileName := fmt.Sprintf("photo_%s_%d.jpg", time.Now().Format("2006-01-02_15-04-05"), photo.GetID())
		file := &tgFile{
			location: location,
			size:     0,
			name:     fileName,
		}
		for _, opt := range opts {
			opt(file)
		}
		return file, nil
	}
	return nil, fmt.Errorf("unsupported media type: %T", media)
}
