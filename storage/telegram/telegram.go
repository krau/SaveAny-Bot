package telegram

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/rs/xid"
	"golang.org/x/time/rate"
)

const (
	// https://core.telegram.org/api/config#upload-max-fileparts-default
	DefaultSplitSize  = 4000 * 524288 // 4000 * 512 KB
	MaxUploadFileSize = 4000 * 524288 // 4000 * 512 KB
)

type Telegram struct {
	config  storconfig.TelegramStorageConfig
	limiter *rate.Limiter
}

func (t *Telegram) Init(ctx context.Context, cfg storconfig.StorageConfig) error {
	telegramConfig, ok := cfg.(*storconfig.TelegramStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast telegram config")
	}
	if err := telegramConfig.Validate(); err != nil {
		return err
	}
	t.config = *telegramConfig
	if t.config.RateLimit <= 0 || t.config.RateBurst <= 0 {
		t.config.RateLimit = 2
		t.config.RateBurst = 1
	}
	t.limiter = rate.NewLimiter(rate.Every(time.Duration(t.config.RateLimit)*time.Second), t.config.RateBurst)
	return nil
}

func (t *Telegram) Type() storenum.StorageType {
	return storenum.Telegram
}

func (t *Telegram) Name() string {
	return t.config.Name
}

func (t *Telegram) Exists(ctx context.Context, storagePath string) bool {
	return false
}

func (t *Telegram) Save(ctx context.Context, r io.Reader, storagePath string) error {
	storagePath = path.Clean(storagePath)
	tctx := tgutil.ExtFromContext(ctx)
	if tctx == nil {
		return fmt.Errorf("failed to get telegram context")
	}
	size := func() int64 {
		if length := ctx.Value(ctxkey.ContentLength); length != nil {
			if l, ok := length.(int64); ok {
				return l
			}
		}
		return -1 // unknown size
	}()
	if t.config.SkipLarge && size > MaxUploadFileSize {
		log.FromContext(ctx).Warnf("Skipping file larger than Telegram limit (%d bytes): %d bytes", MaxUploadFileSize, size)
		return nil
	}
	rs, seekable := r.(io.ReadSeeker)
	splitSize := t.config.SplitSizeMB * 1024 * 1024
	if splitSize <= 0 {
		splitSize = DefaultSplitSize
	}

	if err := t.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit failed: %w", err)
	}

	// 去除前导斜杠并分隔路径, 当 len(parts):
	// ==0, 存储到配置文件中的 chat_id, 随机文件名
	// ==1, 视作只有文件名, 存储到配置文件中的 chat_id
	// ==2, parts[0]: 视作要存储到的 chat_id, parts[1]: filename
	parts := slice.Compact(strings.Split(strings.TrimPrefix(storagePath, "/"), "/"))
	filename := ""
	chatID := t.config.ChatID
	if len(parts) >= 1 {
		filename = parts[len(parts)-1]
	}
	if len(parts) >= 2 && validator.IsAlphaNumeric(parts[0]) {
		cid, err := tgutil.ParseChatID(tctx, parts[0])
		if err != nil {
			// id不合法时使用配置文件中的 chat_id
			log.FromContext(ctx).Warnf("Failed to parse chat ID from path, using configured chat_id: %s", err)
			cid = chatID
		}
		chatID = cid
	}
	upler := uploader.NewUploader(tctx.Raw).
		WithPartSize(tglimit.MaxUploadPartSize).
		WithThreads(dlutil.BestThreads(size, config.C().Threads))
	peer := tryGetInputPeer(tctx, chatID)
	if peer == nil || peer.Zero() {
		return fmt.Errorf("failed to get input peer for chat ID %d", chatID)
	}
	var mtype *mimetype.MIME
	if seekable {
		var err error
		mtype, err = mimetype.DetectReader(rs)
		if err != nil {
			return fmt.Errorf("failed to detect mimetype: %w", err)
		}
		if filename == "" {
			filename = xid.New().String() + mtype.Extension()
		}

		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek reader: %w", err)
		}
	}
	if size > splitSize {
		// large file, use split uploader
		return t.splitUpload(tctx, r, filename, upler, peer, size, splitSize)
	}

	var file tg.InputFileClass
	var err error
	if size <= 0 {
		file, err = upler.FromReader(ctx, filename, r)
	} else {
		file, err = upler.Upload(ctx, uploader.NewUpload(filename, r, size))
	}
	if err != nil {
		return fmt.Errorf("failed to upload file to telegram: %w", err)
	}
	caption := styling.Plain(filename)
	forceFile := t.config.ForceFile

	if mtype != nil && strings.HasPrefix(mtype.String(), "image/") && size >= tglimit.MaxPhotoSize {
		forceFile = true
	}
	doc := message.UploadedDocument(file, caption).
		Filename(filename).
		ForceFile(forceFile)
	if mtype != nil {
		doc = doc.MIME(mtype.String())
	}
	var media message.MediaOption = doc
	if mtype != nil && rs != nil {
		switch mtypeStr := mtype.String(); {
		case strings.HasPrefix(mtypeStr, "video/"):
			media = doc.Video().SupportsStreaming()
			thumb, err := extractThumbFrame(rs)
			if err == nil {
				thumb, err := upler.FromBytes(ctx, "thumb.jpg", thumb)
				if err == nil {
					doc = doc.Thumb(thumb)
				}
			}
			rs.Seek(0, io.SeekStart)
			switch mtypeStr {
			case "video/mp4":
				info, err := getMP4Meta(rs)
				if err != nil {
					// Fallback to ffprobe if gomedia fails (e.g., malformed MP4)
					rs.Seek(0, io.SeekStart)
					info, err = getVideoMetadata(rs)
				}
				if err == nil {
					media = doc.Video().
						Duration(time.Duration(info.Duration)*time.Second).
						Resolution(info.Width, info.Height).
						SupportsStreaming()
				}
			default:
				info, err := getVideoMetadata(rs)
				if err == nil {
					media = doc.Video().
						Duration(time.Duration(info.Duration)*time.Second).
						Resolution(info.Width, info.Height).
						SupportsStreaming()
				}
			}
		case strings.HasPrefix(mtypeStr, "audio/"):
			media = doc.Audio().Title(filename)
		case strings.HasPrefix(mtypeStr, "image/") && !strings.HasSuffix(mtypeStr, "webp"):
			media = message.UploadedPhoto(file, caption)
		}
	}
	sender := tctx.Sender
	_, err = sender.WithUploader(upler).To(peer).Media(ctx, media)
	return err
}

func (t *Telegram) CannotStream() string {
	return "Telegram storage must use a ReaderSeeker"
}

func (t *Telegram) splitUpload(ctx *ext.Context, r io.Reader, filename string, upler *uploader.Uploader, peer tg.InputPeerClass, fileSize, splitSize int64) error {
	tempId := xid.New().String()
	outputBase := filepath.Join(config.C().Temp.BasePath, tempId, strings.Split(filename, ".")[0])
	defer func() {
		// cleanup temp files
		if err := os.RemoveAll(filepath.Join(config.C().Temp.BasePath, tempId)); err != nil {
			log.FromContext(ctx).Warnf("Failed to cleanup temp split files: %s", err)
		}
	}()
	if err := CreateSplitZip(ctx, r, fileSize, filename, outputBase, splitSize); err != nil {
		return fmt.Errorf("failed to create split zip: %w", err)
	}
	matched, err := filepath.Glob(outputBase + ".z*")
	if err != nil {
		return fmt.Errorf("failed to glob split files: %w", err)
	}
	inputFiles := make([]tg.InputFileClass, 0, len(matched))
	for _, partPath := range matched {
		// 串行上传, 不然容易被tg风控
		err = func() error {
			partFile, err := os.Open(partPath)
			if err != nil {
				return fmt.Errorf("failed to open split part %s: %w", partPath, err)
			}
			defer partFile.Close()
			partInfo, err := partFile.Stat()
			if err != nil {
				return fmt.Errorf("failed to stat split part %s: %w", partPath, err)
			}
			partFileSize := partInfo.Size()
			partName := filepath.Base(partPath)
			partInputFile, err := upler.Upload(ctx, uploader.NewUpload(partName, partFile, partFileSize))
			if err != nil {
				return fmt.Errorf("failed to upload split part %s: %w", partPath, err)
			}
			inputFiles = append(inputFiles, partInputFile)
			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to upload split part %s: %w", partPath, err)
		}
	}
	if len(inputFiles) == 1 {
		// only one part, send as normal file
		// shoud not happen as we already check fileSize > splitSize
		doc := message.UploadedDocument(inputFiles[0]).
			Filename(filepath.Base(matched[0])).
			ForceFile(true).
			MIME("application/zip")
		_, err = ctx.Sender.
			WithUploader(upler).
			To(peer).
			Media(ctx, doc)
		return err
	}

	multiMedia := make([]message.MultiMediaOption, 0, len(inputFiles))
	for i, inputFile := range inputFiles {
		doc := message.UploadedDocument(inputFile).
			Filename(filepath.Base(matched[i])).
			MIME("application/zip")
		multiMedia = append(multiMedia, doc)
	}

	sender := ctx.Sender

	if len(multiMedia) <= 10 {
		_, err = sender.WithUploader(upler).
			To(peer).
			Album(ctx, multiMedia[0], multiMedia[1:]...)
		return err
	}

	// more than 10 parts, send in batches, each batch up to 10 parts
	for i := 0; i < len(multiMedia); i += 10 {
		end := min(i+10, len(multiMedia))
		batch := multiMedia[i:end]
		_, err = sender.WithUploader(upler).
			To(peer).
			Album(ctx, batch[0], batch[1:]...)
		if err != nil {
			return fmt.Errorf("failed to send album batch: %w", err)
		}
	}
	return nil

}
