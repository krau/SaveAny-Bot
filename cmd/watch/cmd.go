package watch

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	stortype "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "watch a local directory and auto-upload changed files to storage",
	Long: `Watch a local directory and automatically upload created or modified files
to the specified storage backend, preserving the relative directory structure.

Example:
  saveany-bot watch -p /data/inbox -s mystorage -d backup --recursive`,
	RunE: runWatch,
}

func Register(root *cobra.Command) {
	flags := watchCmd.Flags()
	flags.StringP("path", "p", "", "local directory path to watch")
	watchCmd.MarkFlagRequired("path")
	flags.StringP("storage", "s", "", "storage name to upload to")
	watchCmd.MarkFlagRequired("storage")
	flags.StringP("dir", "d", "", "storage dir to upload to, default is the base_path of the storage")
	flags.BoolP("recursive", "r", false, "watch subdirectories recursively")
	flags.Bool("overwrite", false, "overwrite existing files on storage instead of skipping")
	flags.Bool("initial-scan", false, "upload existing files in the directory on startup")
	flags.Duration("debounce", 2*time.Second, "wait time after the last change before uploading a file")
	flags.Int("upload-workers", 0, "number of concurrent uploads, default is config.workers")
	flags.Duration("retry-delay", 3*time.Second, "delay between upload retries")
	root.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, _ []string) error {
	watchPath, err := cmd.Flags().GetString("path")
	if err != nil {
		return err
	}
	storName, err := cmd.Flags().GetString("storage")
	if err != nil {
		return err
	}
	destDir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	recursive, err := cmd.Flags().GetBool("recursive")
	if err != nil {
		return err
	}
	overwrite, err := cmd.Flags().GetBool("overwrite")
	if err != nil {
		return err
	}
	initialScan, err := cmd.Flags().GetBool("initial-scan")
	if err != nil {
		return err
	}
	debounce, err := cmd.Flags().GetDuration("debounce")
	if err != nil {
		return err
	}
	uploadWorkers, err := cmd.Flags().GetInt("upload-workers")
	if err != nil {
		return err
	}
	retryDelay, err := cmd.Flags().GetDuration("retry-delay")
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	logger := log.FromContext(ctx)

	configFile := config.GetConfigFile(cmd)
	if err := config.Init(ctx, configFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	i18n.Init(config.C().Lang)
	cache.Init()
	database.Init(ctx)

	stor, err := storage.GetStorageByName(ctx, storName)
	if err != nil {
		return fmt.Errorf("failed to get storage %q: %w", storName, err)
	}

	// Telegram storage needs the bot client and its ext context injected into ctx.
	if stor.Type() == stortype.Telegram {
		bot.Init(ctx)
		ctx = tgutil.ExtWithContext(ctx, bot.ExtContext())
	}

	if uploadWorkers < 1 {
		uploadWorkers = config.C().Workers
	}

	uploader := NewUploader(ctx, UploaderOptions{
		Storage:    stor,
		DestDir:    destDir,
		Overwrite:  overwrite,
		Workers:    uploadWorkers,
		Retry:      config.C().Retry,
		RetryDelay: retryDelay,
	})

	watcher, err := NewWatcher(ctx, WatcherOptions{
		Root:      watchPath,
		Recursive: recursive,
		Debounce:  debounce,
		Uploader:  uploader,
	})
	if err != nil {
		uploader.Close()
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	if initialScan {
		watcher.ScanExisting(ctx)
	}

	logger.Infof("watch started: %s -> storage %q dir %q", watchPath, storName, destDir)

	// Run blocks until ctx is cancelled (e.g. SIGINT).
	runErr := watcher.Run(ctx)

	// Wait for in-flight uploads to finish before exiting.
	logger.Info("waiting for in-flight uploads to finish...")
	uploader.Close()
	logger.Info("watch stopped")

	return runErr
}
