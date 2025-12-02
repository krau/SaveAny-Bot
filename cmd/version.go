package cmd

import (
	"fmt"
	"runtime"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/unvgo/ghselfupdate"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number of saveany-bot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("saveany-bot version: %s %s/%s\nBuildTime: %s, Commit: %s\n", config.Version, runtime.GOOS, runtime.GOARCH, config.BuildTime, config.GitCommit)
	},
}

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"up"},
	Short:   "Upgrade saveany-bot to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		v := semver.MustParse(config.Version)
		latest, found, err := ghselfupdate.DetectLatest(config.GitRepo)
		if err != nil {
			fmt.Println("Error occurred while detecting latest version:", err)
			return
		}
		if !found {
			fmt.Println("No releases found")
			return
		}
		if latest.Version.Major != v.Major {
			fmt.Printf("Major version upgrade detected: %s -> %s. Please manually download the latest version and check the migration guide.\n", v, latest.Version)
			return
		}
		if latest.Version.Equals(v) || latest.Version.LT(v) {
			fmt.Println("Current binary is the latest version", config.Version)
			return
		}
		fmt.Printf("Updating to version %s...\n", latest.Version)
		latest, err = ghselfupdate.UpdateSelf(v, config.GitRepo)
		if err != nil {
			fmt.Println("Update failed:", err)
			return
		}
		fmt.Println("Successfully updated to version", latest.Version)
		fmt.Println("Release note:\n", latest.ReleaseNotes)

	},
}

func init() {
	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(upgradeCmd)
}
