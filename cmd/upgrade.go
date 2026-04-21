package cmd

import (
	"fmt"
	"os"

	"github.com/ropean/muze/internal/selfupdate"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade to a newer version",
	Long:  "Download and replace the current binary. Defaults to the latest release; use --version to pin a tag.",
	Run: func(cmd *cobra.Command, _ []string) {
		target, _ := cmd.Flags().GetString("version")

		if target == "" || target == "latest" {
			rel, err := selfupdate.LatestRelease()
			if err != nil {
				writeError("resolve latest version: %s", err)
			}
			target = rel.TagName

			if !selfupdate.IsNewer(target) {
				fmt.Fprintln(os.Stderr, "Already up to date")
				return
			}
		}

		fmt.Fprintf(os.Stderr, "Upgrading %s → %s ...\n", selfupdate.Version, target)

		tmp, err := selfupdate.DownloadAsset(target)
		if err != nil {
			writeError("download: %s", err)
		}
		defer os.Remove(tmp)

		if err := selfupdate.ReplaceBinary(tmp); err != nil {
			writeError("replace binary: %s", err)
		}

		fmt.Fprintf(os.Stderr, "Upgraded to %s\n", target)
	},
}

func init() {
	upgradeCmd.Flags().String("version", "latest", "Target version tag (e.g. v1.0.0)")
}
