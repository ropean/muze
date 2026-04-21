package cmd

import (
	"fmt"
	"os"

	"github.com/ropean/muze/internal/selfupdate"
	"github.com/spf13/cobra"
)

var checkUpdateCmd = &cobra.Command{
	Use:   "check-update",
	Short: "Check if a newer version is available",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Fprintf(os.Stderr, "Current version: %s\n", selfupdate.Version)

		rel, err := selfupdate.LatestRelease()
		if err != nil {
			writeError("check failed: %s", err)
		}

		if selfupdate.IsNewer(rel.TagName) {
			fmt.Fprintf(os.Stderr, "New version available: %s\n", rel.TagName)
			fmt.Fprintf(os.Stderr, "Run `muze upgrade` to update\n")
			fmt.Fprintf(os.Stderr, "Release: %s\n", rel.HTMLURL)
		} else {
			fmt.Fprintln(os.Stderr, "Already up to date")
		}
	},
}
