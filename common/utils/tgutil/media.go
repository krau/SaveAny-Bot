package tgutil

import (
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/tg"
)

func GetMediaFileName(media tg.MessageMediaClass) (string, error) {
	switch v := media.(type) {
	case *tg.MessageMediaPhoto:
		f, ok := v.Photo.AsNotEmpty()
		if !ok {
			return "", fmt.Errorf("unknown type media: %T", media)
		}
		return fmt.Sprintf("%d.png", f.ID), nil
	case *tg.MessageMediaDocument:
		f, ok := v.Document.AsNotEmpty()
		if !ok {
			return "", fmt.Errorf("unknown type media: %T", media)
		}
		fileName := ""
		for _, attribute := range f.Attributes {
			if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
				fileName = name.GetFileName()
				break
			}
		}
		if fileName == "" {
			mmt := mimetype.Lookup(f.GetMimeType())
			if mmt != nil {
				fileName = fmt.Sprintf("%d.%s", f.GetID(), mmt.Extension())
			}
		}
		return fileName, nil
	default:
		return "", fmt.Errorf("unsupported type media: %T", media)
	}
}
