package types

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gotd/td/tg"
)

type Task struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	Error       error
	Status      TaskStatus
	StorageName string
	StoragePath string
	StartTime   time.Time

	File          *File
	FileMessageID int
	FileChatID    int64

	IsTelegraph  bool
	TelegraphURL string

	// to track the reply message
	ReplyMessageID int
	ReplyChatID    int64
	UserID         int64
}

func (t Task) Key() string {
	if t.IsTelegraph {
		return hashStr(t.TelegraphURL)
	}
	return fmt.Sprintf("%d:%d", t.FileChatID, t.FileMessageID)
}

func (t Task) String() string {
	if t.IsTelegraph {
		return fmt.Sprintf("[telegraph]:%s", t.TelegraphURL)
	}
	return fmt.Sprintf("[%d:%d]:%s", t.FileChatID, t.FileMessageID, t.File.FileName)
}

func (t Task) FileName() string {
	if t.IsTelegraph {
		tgphPath := strings.Split(t.TelegraphURL, "/")[len(strings.Split(t.TelegraphURL, "/"))-1]
		tgphPathUnescaped, err := url.PathUnescape(tgphPath)
		if err != nil {
			return tgphPath
		}
		return tgphPathUnescaped
	}
	return t.File.FileName
}

type File struct {
	Location tg.InputFileLocationClass
	FileSize int64
	FileName string
}

func (f File) Hash() string {
	locationBytes := []byte(f.Location.String())
	fileSizeBytes := []byte(fmt.Sprintf("%d", f.FileSize))
	fileNameBytes := []byte(f.FileName)

	structBytes := append(locationBytes, fileSizeBytes...)
	structBytes = append(structBytes, fileNameBytes...)

	hash := md5.New()
	hash.Write(structBytes)
	hashBytes := hash.Sum(nil)

	return hex.EncodeToString(hashBytes)
}
