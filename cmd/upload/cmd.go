package upload

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/boot"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	stortype "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "upload local files to storage",
	SilenceUsage: true,
	RunE:         Upload,
}

func Register(root *cobra.Command) {
	uploadCmd.Flags().StringP("file", "f", "", "file path to upload")
	uploadCmd.MarkFlagRequired("file")
	uploadCmd.Flags().StringP("storage", "s", "", "storage name to upload to")
	uploadCmd.MarkFlagRequired("storage")
	uploadCmd.Flags().StringP("dir", "d", "", "storage dir to upload to, default is the base_path of the storage")
	uploadCmd.Flags().Bool("no-progress", false, "disable progress bar")
	root.AddCommand(uploadCmd)
}

func Upload(cmd *cobra.Command, args []string) error {
	storname, err := cmd.Flags().GetString("storage")
	if err != nil {
		return err
	}
	fp, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	dirPath, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	noProgress, err := cmd.Flags().GetBool("no-progress")
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	logger := log.FromContext(ctx)
	opts := boot.Parse(cmd)
	opts.InitWeb = false
	if _, err := boot.Init(ctx, opts); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	stor, err := storage.GetStorageByName(ctx, storname)
	if err != nil {
		return fmt.Errorf("failed to get storage %q: %w", storname, err)
	}

	switch stor.Type() {
	case stortype.Telegram:
		if err := bot.Start(ctx); err != nil {
			return fmt.Errorf("failed to start bot: %w", err)
		}
		defer bot.Stop()
	}

	fileInfo, err := os.Stat(filepath.Clean(fp))
	if err != nil {
		return fmt.Errorf("failed to stat file %q: %w", fp, err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("%q is a directory", fp)
	}

	uploadPath := path.Join(dirPath, fileInfo.Name())
	ctx = tgutil.ExtWithContext(ctx, bot.ExtContext())

	var progressUI *UploadProgress
	if !noProgress && fileInfo.Size() > 0 {
		progressUI = NewUploadProgress(fileInfo.Name(), fileInfo.Size())
		progressUI.Start()
	}
	onProgress := func(uploaded, total int64) {
		if progressUI != nil && total > 0 {
			progressUI.UpdateProgress(float64(uploaded) / float64(total))
		}
	}
	logger.Info("Uploading file...", "file", fp, "to", storname, "as", uploadPath)
	err = upload.SaveFile(ctx, filepath.Clean(fp), uploadPath, upload.Options{
		Storage:    stor,
		Retries:    max(config.C().Retry, 1),
		RetryDelay: 3 * time.Second,
		OnProgress: onProgress,
	})
	if err != nil {
		if progressUI != nil {
			progressUI.SetError(err)
		}
		return err
	}
	if progressUI != nil {
		progressUI.Done()
	}
	logger.Info("File uploaded successfully")
	return nil
}
