package cmd

import (
	"fmt"
	"runtime"

	"github.com/krau/SaveAny-Bot/common"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number of saveany-bot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("saveany-bot version: %s %s/%s\nBuildTime: %s, Commit: %s\n", common.Version, runtime.GOOS, runtime.GOARCH, common.BuildTime, common.GitCommit)
	},
}
