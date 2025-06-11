package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "saveany-bot",
	Short: "saveany-bot",
	Run:   Run,
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
	}
}
