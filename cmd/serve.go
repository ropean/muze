package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ropean/muze/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server mode",
	Run: func(cmd *cobra.Command, _ []string) {
		port, _ := cmd.Flags().GetInt("port")
		addr := fmt.Sprintf(":%d", port)

		routes := server.Routes()

		// Build rows: [method, path, param0, param1, ...]
		rows := make([][]string, len(routes))
		for i, r := range routes {
			rows[i] = append([]string{r.Method, r.Path}, r.Params...)
		}

		// Compute per-column max width (all ASCII, so len == visual width).
		maxCols := 0
		for _, row := range rows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}
		colW := make([]int, maxCols)
		for _, row := range rows {
			for j, cell := range row {
				if len(cell) > colW[j] {
					colW[j] = len(cell)
				}
			}
		}

		fmt.Fprintf(os.Stderr, "muze listening on http://localhost%s\n\n", addr)
		for _, row := range rows {
			fmt.Fprint(os.Stderr, "  ")
			for j, cell := range row {
				if j > 0 {
					fmt.Fprint(os.Stderr, "  ")
				}
				if j < len(row)-1 {
					fmt.Fprintf(os.Stderr, "%-*s", colW[j], cell)
				} else {
					fmt.Fprint(os.Stderr, cell) // last column: no trailing pad
				}
			}
			fmt.Fprintln(os.Stderr)
		}
		fmt.Fprintln(os.Stderr)

		reg := registry()
		if err := http.ListenAndServe(addr, server.New(reg)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	serveCmd.Flags().Int("port", 8010, "Port to listen on")
}
