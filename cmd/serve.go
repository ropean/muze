package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ropean/music-dl-cn/internal/api"
	"github.com/ropean/music-dl-cn/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server mode",
	Run: func(cmd *cobra.Command, _ []string) {
		port, _ := cmd.Flags().GetInt("port")
		addr := fmt.Sprintf(":%d", port)
		fmt.Fprintf(os.Stderr, "music-dl server listening on %s\n", addr)
		handler := server.New(api.NewRegistry())
		if err := http.ListenAndServe(addr, handler); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	serveCmd.Flags().Int("port", 8080, "Port to listen on")
}
