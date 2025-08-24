package telegram

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/constant"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/rs/xid"
	"golang.org/x/time/rate"
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

func (t *Telegram) JoinStoragePath(p string) string {
	return path.Clean(p)
}

func (t *Telegram) Exists(ctx context.Context, storagePath string) bool {
	return false
}

func (t *Telegram) Save(ctx context.Context, r io.Reader, storagePath string) error {
	if err := t.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit failed: %w", err)
	}
	rs, ok := r.(io.ReadSeeker)
	if !ok || rs == nil {
		return fmt.Errorf("reader must implement io.ReadSeeker")
	}
	tctx := tgutil.ExtFromContext(ctx)
	if tctx == nil {
		return fmt.Errorf("failed to get telegram context")
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
		} else {
			if cid > constant.MaxTDLibChannelID || cid > constant.MaxTDLibChatID || cid > constant.MaxTDLibUserID {
				cid = chatID
			}
		}
		chatID = cid
	}
	mtype, err := mimetype.DetectReader(rs)
	if err != nil {
		return fmt.Errorf("failed to detect mimetype: %w", err)
	}
	if filename == "" {
		filename = xid.New().String() + mtype.Extension()
	}

	if chatID < 0 {
		chatID = chatID - constant.ZeroTDLibChannelID
	}
	peer := tctx.PeerStorage.GetInputPeerById(chatID)
	if peer == nil {
		return fmt.Errorf("failed to get input peer for chat ID %d", chatID)
	}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek reader: %w", err)
	}
	upler := uploader.NewUploader(tctx.Raw).
		WithPartSize(tglimit.MaxUploadPartSize).
		WithThreads(config.C().Threads)

	var file tg.InputFileClass
	size := func() int64 {
		if length := ctx.Value(ctxkey.ContentLength); length != nil {
			if l, ok := length.(int64); ok {
				return l
			}
		}
		return -1 // unknown size
	}()
	if size < 0 {
		file, err = upler.FromReader(ctx, filename, rs)
	} else {
		file, err = upler.Upload(ctx, uploader.NewUpload(filename, rs, size))
	}
	if err != nil {
		return fmt.Errorf("failed to upload file to telegram: %w", err)
	}
	caption := styling.Plain(filename)
	forceFile := t.config.ForceFile
	if strings.HasPrefix(mtype.String(), "image/") && size >= tglimit.MaxPhotoSize {
		forceFile = true
	}
	docb := message.UploadedDocument(file, caption).
		Filename(filename).
		ForceFile(forceFile).
		MIME(mtype.String())

	var media message.MediaOption = docb

	switch mtypeStr := mtype.String(); {
	case strings.HasPrefix(mtypeStr, "video/"):
		media = docb.Video().SupportsStreaming()
	case strings.HasPrefix(mtypeStr, "audio/"):
		media = docb.Audio().Title(filename)
	case strings.HasPrefix(mtypeStr, "image/") && !strings.HasSuffix(mtypeStr, "webp"):
		media = message.UploadedPhoto(file, caption)
	}
	sender := tctx.Sender
	_, err = sender.WithUploader(upler).To(peer).Media(ctx, media)
	return err
}

func (t *Telegram) CannotStream() string {
	return "Telegram storage must use a ReaderSeeker"
}
