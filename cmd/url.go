package cmd

import (
	"github.com/ropean/muze/internal/api"
	"github.com/spf13/cobra"
)

var urlCmd = &cobra.Command{
	Use:   "url <source> <id>",
	Short: "Resolve playback URL for a track",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source, id := args[0], args[1]
		quality, _ := cmd.Flags().GetString("quality")
		result, err := registry().GetURL(source, id, api.URLOptions{Quality: quality})
		if err != nil {
			writeError(err.Error())
		}
		writeJSON(result)
	},
}

func init() {
	urlCmd.Flags().String("quality", "", "Audio quality: flac|320k|128k (default: 320k)")
}
