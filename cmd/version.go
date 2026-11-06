package cmd

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/krau/SaveAny-Bot/common/selfupdate"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/unvgo/ghselfupdate"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number of saveany-bot",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("saveany-bot version: %s %s/%s\nBuildTime: %s, Commit: %s\n", config.Version, runtime.GOOS, runtime.GOARCH, config.BuildTime, config.GitCommit)
		return nil
	},
}

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"up"},
	Short:   "Upgrade saveany-bot to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := semver.Parse(config.Version)
		if err != nil {
			return fmt.Errorf("current build has no release version: %w", err)
		}
		if err := selfupdate.Updatable(); err != nil {
			return fmt.Errorf("this build cannot update itself: %w", err)
		}
		latest, found, err := ghselfupdate.DetectLatest(config.GitRepo)
		if err != nil {
			return fmt.Errorf("detect latest release: %w", err)
		}
		if !found {
			return errors.New("no releases found")
		}
		if err := selfupdate.CheckTarget(v, latest.Version); err != nil {
			switch {
			case errors.Is(err, selfupdate.ErrMajorUpgrade):
				return fmt.Errorf("major upgrade %s -> %s requires a manual download and migration review", v, latest.Version)
			case errors.Is(err, selfupdate.ErrAlreadyLatest):
				fmt.Println("Current binary is the latest version", config.Version)
				return nil
			default:
				return err
			}
		}
		fmt.Printf("Updating to version %s...\n", latest.Version)
		latest, err = selfupdate.Update(v)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Println("Successfully updated to version", latest.Version)
		fmt.Println("Release note:\n", latest.ReleaseNotes)
		fmt.Println("Restart saveany-bot to use it.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(upgradeCmd)
}
