package cmd

import (
	"fmt"
	"os"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/downloader"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <source> <id>",
	Short: "Download a track by source and ID",
	Long: `Resolve a playback URL and download the track to a local file.

The default filename is "<title> - <artist>.mp3". Use --out to specify
a custom path, or --title/--artist to control the default name.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source, id := args[0], args[1]
		outPath, _ := cmd.Flags().GetString("out")
		title, _ := cmd.Flags().GetString("title")
		artist, _ := cmd.Flags().GetString("artist")
		force, _ := cmd.Flags().GetBool("force")

		reg := api.NewRegistry()
		result, err := reg.GetURL(source, id, api.URLOptions{})
		if err != nil {
			writeError("resolve url: %s", err)
		}

		if outPath == "" {
			if title == "" {
				title = id
			}
			outPath = downloader.DefaultFilename(title, artist)
		}

		fmt.Fprintf(os.Stderr, "Downloading to %s ...\n", outPath)

		err = downloader.Download(downloader.Options{
			URL:     result.URL,
			OutPath: outPath,
			Force:   force,
			OnProgress: func(current, total int64) {
				if total > 0 {
					pct := float64(current) / float64(total) * 100
					fmt.Fprintf(os.Stderr, "\r  %.1f%% (%s / %s)",
						pct, downloader.FormatBytes(current), downloader.FormatBytes(total))
				} else {
					fmt.Fprintf(os.Stderr, "\r  %s downloaded", downloader.FormatBytes(current))
				}
			},
		})
		if err != nil {
			writeError("download: %s", err)
		}

		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Saved: %s\n", outPath)
	},
}

func init() {
	downloadCmd.Flags().String("out", "", "Output file path (default: <title> - <artist>.mp3)")
	downloadCmd.Flags().String("title", "", "Track title (for default filename)")
	downloadCmd.Flags().String("artist", "", "Artist name (for default filename)")
	downloadCmd.Flags().Bool("force", false, "Overwrite existing file")
}
