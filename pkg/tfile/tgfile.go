package tfile

import (
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

type TGFileMessage interface {
	TGFile
	Message() *tg.Message
}

type tgFile struct {
	location tg.InputFileLocationClass
	size     int64
	name     string
	message  *tg.Message
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

func (f *tgFile) Message() *tg.Message {
	return f.message
}

func NewTGFile(location tg.InputFileLocationClass, size int64, name string,
	opts ...TGFileOptions,
) TGFile {
	f := &tgFile{
		location: location,
		size:     size,
		name:     name,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func FromMedia(media tg.MessageMediaClass, opts ...TGFileOptions) (TGFile, error) {
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

func FromMediaMessage(media tg.MessageMediaClass, msg *tg.Message, opts ...TGFileOptions) (TGFileMessage, error) {
	file, err := FromMedia(media, opts...)
	if err != nil {
		return nil, err
	}
	return &tgFile{
		location: file.Location(),
		size:     file.Size(),
		name:     file.Name(),
		message:  msg,
	}, nil
}
