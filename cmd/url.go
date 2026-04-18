package cmd

import (
	"github.com/ropean/music-dl-cn/internal/api"
	"github.com/spf13/cobra"
)

var urlCmd = &cobra.Command{
	Use:   "url <source> <id>",
	Short: "Resolve playback URL for a track",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source, id := args[0], args[1]
		result, err := api.NewRegistry().GetURL(source, id)
		if err != nil {
			writeError(err.Error())
		}
		writeJSON(result)
	},
}
