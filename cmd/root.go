package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "saveany-bot",
	Short: "saveany-bot",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

func Execute() {
	rootCmd.Execute()
}
