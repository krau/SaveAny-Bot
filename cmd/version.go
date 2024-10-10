package cmd

import (
	"fmt"
	"runtime"

	"github.com/krau/SaveAny-Bot/common"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/blang/semver"
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

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"up"},
	Short:   "Upgrade saveany-bot to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		v := semver.MustParse(common.Version)
		latest, err := selfupdate.UpdateSelf(v, "krau/SaveAny-Bot")
		if err != nil {
			fmt.Println("Binary update failed:", err)
			return
		}
		if latest.Version.Equals(v) {
			fmt.Println("Current binary is the latest version", common.Version)
		} else {
			fmt.Println("Successfully updated to version", latest.Version)
			fmt.Println("Release note:\n", latest.ReleaseNotes)
		}
	},
}

func init() {
	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(upgradeCmd)
}
