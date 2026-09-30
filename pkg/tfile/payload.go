package tfile

import (
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

// Payloadable is implemented by TGFile implementations that can serialize
// themselves for task recovery.
type Payloadable interface {
	Payload() FilePayload
}

// FilePayloadOf returns the serializable form of f.
func FilePayloadOf(f TGFile) (FilePayload, bool) {
	p, ok := f.(Payloadable)
	if !ok {
		return FilePayload{}, false
	}
	return p.Payload(), true
}

// FilePayload is the minimal serializable representation of a TGFile,
// used to rebuild tasks after a process restart.
type FilePayload struct {
	Kind          string `json:"kind"` // "document" | "photo"
	ID            int64  `json:"id"`
	AccessHash    int64  `json:"access_hash"`
	FileReference []byte `json:"file_reference"`
	ThumbSize     string `json:"thumb_size"`
	Size          int64  `json:"size"`
	Name          string `json:"name"`
}

// Payload returns the serializable representation of the file.
func (f *tgFile) Payload() FilePayload {
	p := FilePayload{
		Size: f.size,
		Name: f.name,
	}
	switch loc := f.location.(type) {
	case *tg.InputDocumentFileLocation:
		p.Kind = "document"
		p.ID = loc.ID
		p.AccessHash = loc.AccessHash
		p.FileReference = loc.FileReference
		p.ThumbSize = loc.ThumbSize
	case *tg.InputPhotoFileLocation:
		p.Kind = "photo"
		p.ID = loc.ID
		p.AccessHash = loc.AccessHash
		p.FileReference = loc.FileReference
		p.ThumbSize = loc.ThumbSize
	}
	return p
}

// Location rebuilds the Telegram file location from the payload.
func (p FilePayload) Location() tg.InputFileLocationClass {
	switch p.Kind {
	case "photo":
		return &tg.InputPhotoFileLocation{
			ID:            p.ID,
			AccessHash:    p.AccessHash,
			FileReference: p.FileReference,
			ThumbSize:     p.ThumbSize,
		}
	default:
		return &tg.InputDocumentFileLocation{
			ID:            p.ID,
			AccessHash:    p.AccessHash,
			FileReference: p.FileReference,
			ThumbSize:     p.ThumbSize,
		}
	}
}

// FileFromPayload rebuilds a TGFile from its serialized payload.
func FileFromPayload(p FilePayload, dler downloader.Client) TGFile {
	return &tgFile{
		location: p.Location(),
		dler:     dler,
		size:     p.Size,
		name:     p.Name,
	}
}
