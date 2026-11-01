package cmd

import (
	"context"
	"os"

	"github.com/krau/SaveAny-Bot/boot"
	"github.com/krau/SaveAny-Bot/cmd/upload"
	"github.com/krau/SaveAny-Bot/cmd/watch"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "saveany-bot",
	Short: "saveany-bot",
	Run:   Run,
}

func init() {
	boot.RegisterFlags(rootCmd)
	upload.Register(rootCmd)
	watch.Register(rootCmd)
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
