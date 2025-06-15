package telegram

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
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
	peer := tctx.PeerStorage.GetInputPeerById(t.config.ChatID)
	if peer == nil {
		return fmt.Errorf("failed to get input peer for chat ID %d", t.config.ChatID)
	}
	mtype, err := mimetype.DetectReader(rs)
	if err != nil {
		return fmt.Errorf("failed to detect mimetype: %w", err)
	}
	filename := path.Base(storagePath)
	if filename == "" {
		filename = xid.New().String() + mtype.Extension()
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek reader: %w", err)
	}
	upler := uploader.NewUploader(tctx.Raw).
		WithPartSize(tglimit.MaxUploadPartSize).
		WithThreads(config.Cfg.Threads)

	file, err := upler.FromReader(ctx, filename, rs)
	if err != nil {
		return fmt.Errorf("failed to upload file to telegram: %w", err)
	}

	caption := styling.Plain(filename)
	docb := message.UploadedDocument(file, caption).
		Filename(filename).
		ForceFile(true).
		MIME(mtype.String())

	var mediaOpt message.MediaOption = docb
	sender := tctx.Sender
	_, err = sender.WithUploader(upler).To(peer).Media(ctx, mediaOpt)
	return err
}

func (t *Telegram) CannotStream() string {
	return "Telegram storage must use a ReaderSeeker"
}
