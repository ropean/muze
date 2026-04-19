package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:               "music-provider-cn",
	Short:             "Cross-platform music search and download CLI",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

// Execute is the entry point called from main.
func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	root.AddCommand(searchCmd, urlCmd, serveCmd)
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
