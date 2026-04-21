package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:               "muze [keyword]",
	Short:             "Cross-platform music search and download CLI",
	Long:              "Run without a subcommand to enter interactive mode: search, select, and batch-download tracks.",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := ""
		if len(args) > 0 {
			keyword = args[0]
		}
		dir, _ := cmd.Flags().GetString("dir")
		return runInteractive(keyword, dir)
	},
}

// Execute is the entry point called from main.
func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	root.Flags().String("dir", "", "Download directory (default: ./downloads/<keyword>)")
	root.AddCommand(searchCmd, urlCmd, downloadCmd, serveCmd, versionCmd, checkUpdateCmd, upgradeCmd)
}

// writeJSON writes v as indented JSON to stdout.
func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		writeError(err.Error())
	}
}

// writeError writes an error JSON envelope to stdout and exits non-zero.
func writeError(msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	b, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintln(os.Stdout, string(b))
	os.Exit(1)
}
