// Package taskspec holds the serialized creation parameters of tasks. They
// are stored with the task record so a queued task can be rebuilt after a
// restart.
package taskspec

import (
	"time"

	"github.com/krau/SaveAny-Bot/pkg/parser"
)

// TGFileItem identifies one Telegram file and its destination.
type TGFileItem struct {
	ChatID    int64  `json:"chat_id"`
	MessageID int    `json:"message_id"`
	Name      string `json:"name"`
	Storage   string `json:"storage"`
	StorageID uint   `json:"storage_id,omitempty"`
	Path      string `json:"path"`
}

// TGFile is used by both single-file and batch Telegram tasks.
type TGFile struct {
	Items     []TGFileItem `json:"items"`
	Overwrite bool         `json:"overwrite,omitempty"`
}

type DirectLinks struct {
	URLs []string `json:"urls"`
	// Path is the destination directory. A pointer distinguishes "root" from
	// an old record that has no persisted path.
	Path *string `json:"path,omitempty"`
}

type YTDLP struct {
	URLs  []string `json:"urls"`
	Flags []string `json:"flags,omitempty"`
}

type Aria2 struct {
	GID  string   `json:"gid,omitempty"`
	URIs []string `json:"uris"`
}

type Parsed struct {
	Item *parser.Item `json:"item"`
}

type Telegraph struct {
	PhPath string   `json:"ph_path"`
	Pics   []string `json:"pics"`
}

// TransferItem describes one file of a storage-to-storage transfer.
type TransferItem struct {
	SourceStorage   string    `json:"source_storage"`
	SourceStorageID uint      `json:"source_storage_id,omitempty"`
	SourcePath      string    `json:"source_path"`
	TargetStorage   string    `json:"target_storage"`
	TargetStorageID uint      `json:"target_storage_id,omitempty"`
	TargetPath      string    `json:"target_path"`
	Name            string    `json:"name"`
	Size            int64     `json:"size"`
	ModTime         time.Time `json:"mod_time,omitempty"`
}

type Transfer struct {
	Items []TransferItem `json:"items"`
}
