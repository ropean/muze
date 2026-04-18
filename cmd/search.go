package cmd

import (
	"github.com/ropean/music-dl-cn/internal/api"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search for tracks",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		sources, _ := cmd.Flags().GetString("sources")

		result, err := api.NewRegistry().Search(api.SearchRequest{
			Keyword: args[0],
			Sources: sources,
			Page:    page,
			Limit:   limit,
		})
		if err != nil {
			writeError(err.Error())
		}
		writeJSON(result)
	},
}

func init() {
	searchCmd.Flags().Int("page", 1, "Page number (≥ 1)")
	searchCmd.Flags().Int("limit", 30, "Results per page (1–100)")
	searchCmd.Flags().String("sources", "", "Comma-separated sources (default: all)")
}
