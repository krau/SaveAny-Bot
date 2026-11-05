package tfile

import "github.com/gotd/td/tg"

// FileIdentity is the stable part of a Telegram file location. It is used to
// reject resume state left by a different file.
type FileIdentity struct {
	Kind       string
	ID         int64
	AccessHash int64
}

// Identity returns the identity of a Telegram file location. Unknown location
// types return an empty Kind and are not used for resume validation.
func Identity(location tg.InputFileLocationClass) FileIdentity {
	switch loc := location.(type) {
	case *tg.InputDocumentFileLocation:
		return FileIdentity{Kind: "document", ID: loc.ID, AccessHash: loc.AccessHash}
	case *tg.InputPhotoFileLocation:
		return FileIdentity{Kind: "photo", ID: loc.ID, AccessHash: loc.AccessHash}
	default:
		return FileIdentity{}
	}
}

func (i FileIdentity) Matches(other FileIdentity) bool {
	return i.Kind == other.Kind && i.ID == other.ID && i.AccessHash == other.AccessHash
}
