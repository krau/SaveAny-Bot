package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	stortype "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload local files to storage",
	RunE:  Upload,
}

func init() {
	uploadCmd.Flags().StringP("file", "f", "", "file path to upload")
	uploadCmd.MarkFlagRequired("file")
	uploadCmd.Flags().StringP("storage", "s", "", "storage name to upload to")
	uploadCmd.MarkFlagRequired("storage")
	uploadCmd.Flags().StringP("dir", "d", "", "storage dir to upload to, default is the base_path of the storage")
	rootCmd.AddCommand(uploadCmd)
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

	ctx := cmd.Context()
	log := log.FromContext(ctx)
	configFile := config.GetConfigFile(cmd)
	if err := config.Init(ctx, configFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cache.Init()
	database.Init(ctx)

	stor, err := storage.GetStorageByName(ctx, storname)
	if err != nil {
		log.Fatal("Failed to get storage", "error", err)
	}

	switch stor.Type() {
	case stortype.Telegram:
		bot.Init(ctx)
	default:
		// placeholder for other storage types that may need special initialization
	}

	file, err := os.Open(filepath.Clean(fp))
	if err != nil {
		log.Fatal("Failed to open file", "error", err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal("Failed to get file info", "error", err)
	}
	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	uploadPath := stor.JoinStoragePath(path.Join(dirPath, fileName))

	ctx = context.WithValue(ctx, ctxkey.ContentLength, fileSize)
	ctx = tgutil.ExtWithContext(ctx, bot.ExtContext())

	log.Info("Uploading file...", "file", fp, "to", storname, "as", uploadPath)
	if err := stor.Save(ctx, file, uploadPath); err != nil {
		log.Fatal("Failed to upload file", "error", err)
	}
	log.Info("File uploaded successfully")
	return nil
}
