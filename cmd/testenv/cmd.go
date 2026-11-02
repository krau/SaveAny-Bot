package testenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/krau/SaveAny-Bot/app"
	"github.com/krau/SaveAny-Bot/boot"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/settings"
)

var testenvCmd = &cobra.Command{
	Use:   "testenv",
	Short: "run a seeded local environment for end-to-end testing",
	Long: `Run the web UI and task pipeline against a disposable, seeded data directory.

No Telegram credentials are needed. The command creates an admin account, a
local storage and a whitelisted user, then prints the login details and a
sample URL that can be used to create a direct-links task.`,
	RunE: run,
}

func Register(root *cobra.Command) {
	flags := testenvCmd.Flags()
	flags.String("admin-password", "", "web admin password (default saveany-test)")
	flags.String("storage-name", "test-local", "name of the seeded local storage")
	flags.Int64("user-chat-id", 10001, "chat ID of the seeded whitelisted user")
	flags.Bool("keep", false, "keep the generated data directory after exit")
	root.AddCommand(testenvCmd)
}

func run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := boot.Parse(cmd)
	opts.InitWeb = true

	autoDir := false
	if !cmd.Flags().Changed("data-dir") && strings.TrimSpace(os.Getenv("SAVEANY_DATA_DIR")) == "" {
		dir, err := os.MkdirTemp("", "saveany-testenv-")
		if err != nil {
			return fmt.Errorf("create test data directory: %w", err)
		}
		opts.DataDir = dir
		opts.DBPath = filepath.Join(dir, "saveany.db")
		autoDir = true
	}

	keep, _ := cmd.Flags().GetBool("keep")
	if autoDir && !keep {
		defer func() {
			if err := os.RemoveAll(opts.DataDir); err != nil {
				log.Warn("Failed to remove test data directory", "path", opts.DataDir, "error", err)
			}
		}()
	}

	password, _ := cmd.Flags().GetString("admin-password")
	if password == "" {
		password = strings.TrimSpace(os.Getenv("SAVEANY_ADMIN_PASSWORD"))
	}
	if password == "" {
		password = "saveany-test"
	}
	if err := os.Setenv("SAVEANY_ADMIN_PASSWORD", password); err != nil {
		return fmt.Errorf("set admin password: %w", err)
	}

	res, err := boot.Init(ctx, opts)
	if err != nil {
		return fmt.Errorf("initialize test environment: %w", err)
	}

	storageName, _ := cmd.Flags().GetString("storage-name")
	chatID, _ := cmd.Flags().GetInt64("user-chat-id")
	downloadsDir := filepath.Join(opts.DataDir, "downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return fmt.Errorf("create downloads directory: %w", err)
	}
	record, err := res.Settings.CreateStorage(ctx, settings.StorageInput{
		Name:   storageName,
		Type:   "local",
		Enable: true,
		Config: map[string]any{"base_path": downloadsDir},
	})
	if err != nil && !errors.Is(err, settings.ErrStorageExists) {
		return fmt.Errorf("seed local storage: %w", err)
	}
	if record == nil {
		records, err := database.ListStorageRecords(ctx)
		if err != nil {
			return fmt.Errorf("find seeded storage: %w", err)
		}
		for i := range records {
			if records[i].Name == storageName {
				record = &records[i]
				break
			}
		}
		if record == nil {
			return fmt.Errorf("seeded storage %q not found", storageName)
		}
	}

	if _, err := database.GetUserByChatID(ctx, chatID); err != nil {
		if !database.IsNotFound(err) {
			return fmt.Errorf("find seeded user: %w", err)
		}
		if err := database.CreateUser(ctx, chatID); err != nil {
			return fmt.Errorf("create seeded user: %w", err)
		}
	}
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load seeded user: %w", err)
	}
	user.Enabled = true
	user.DefaultStorageID = record.ID
	if err := database.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update seeded user: %w", err)
	}
	if err := res.Settings.ReloadConfig(ctx); err != nil {
		return fmt.Errorf("reload settings: %w", err)
	}

	sampleServer, sampleURL, err := serveSample(ctx, opts.DataDir)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = sampleServer.Shutdown(shutdownCtx)
	}()

	application := app.New(ctx, res.Settings, app.Options{
		DataDir: opts.DataDir,
		DBPath:  opts.DBPath,
		Listen:  res.Listen,
		NoWeb:   res.NoWeb,
	})
	if err := application.Start(); err != nil {
		return fmt.Errorf("start test environment: %w", err)
	}
	defer application.Stop()

	fmt.Println("Test environment ready")
	fmt.Printf("  Web UI:      %s\n", webURL(res.Listen))
	fmt.Printf("  Password:    %s\n", password)
	fmt.Printf("  Data dir:    %s\n", opts.DataDir)
	storagePath := downloadsDir
	if record.ConfigJSON != "" {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(record.ConfigJSON), &cfg); err == nil {
			if path, ok := cfg["base_path"].(string); ok && path != "" {
				storagePath = path
			}
		}
	}
	fmt.Printf("  Storage:     %s -> %s\n", record.Name, storagePath)
	fmt.Printf("  Test user:   %d\n", chatID)
	fmt.Printf("  Sample URL:  %s\n", sampleURL)
	fmt.Println("Press Ctrl+C to stop.")

	<-ctx.Done()
	log.Info("Stopping test environment")
	return nil
}

func serveSample(_ context.Context, dataDir string) (*http.Server, string, error) {
	dir := filepath.Join(dataDir, "sample")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", fmt.Errorf("create sample directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte("SaveAny-Bot end-to-end sample\n"), 0o644); err != nil {
		return nil, "", fmt.Errorf("write sample file: %w", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("listen for sample server: %w", err)
	}
	server := &http.Server{Handler: http.FileServer(http.Dir(dir)), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Sample server error", "error", err)
		}
	}()
	return server, "http://" + ln.Addr().String() + "/sample.txt", nil
}

func webURL(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "http://" + listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}
