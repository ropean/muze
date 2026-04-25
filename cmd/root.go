package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/config"
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
		dir, _     := cmd.Flags().GetString("dir")
		theme, _   := cmd.Flags().GetString("theme")
		quality, _ := cmd.Flags().GetString("quality")
		return runInteractive(keyword, dir, theme, quality)
	},
}

// Execute is the entry point called from main.
func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	root.Flags().String("dir", "", "Download directory (default: ~/Downloads/<keyword>)")
	root.Flags().String("theme", "", "UI theme: base16|tech|charm|dracula|catppuccin (saved to config)")
	root.Flags().String("quality", "", "Audio quality: flac|320k|128k (default: 320k)")
	root.AddCommand(searchCmd, urlCmd, downloadCmd, configCmd, serveCmd, versionCmd, checkUpdateCmd, upgradeCmd)

	// Move -h/--help to the top of every flag block.
	// cobra inherits the template from root, so this applies to all subcommands.
	cobra.AddTemplateFunc("helpFlagFirst", helpFlagFirst)
	tmpl := root.UsageTemplate()
	tmpl = strings.ReplaceAll(tmpl,
		".LocalFlags.FlagUsages | trimTrailingWhitespaces",
		".LocalFlags.FlagUsages | helpFlagFirst | trimTrailingWhitespaces")
	tmpl = strings.ReplaceAll(tmpl,
		".InheritedFlags.FlagUsages | trimTrailingWhitespaces",
		".InheritedFlags.FlagUsages | helpFlagFirst | trimTrailingWhitespaces")
	root.SetUsageTemplate(tmpl)
}

// helpFlagFirst moves the -h, --help line to the top of a pflag usage block.
// pflag computes column alignment across all flags before assembling lines,
// so reordering lines does not break indentation.
func helpFlagFirst(usage string) string {
	lines := strings.Split(usage, "\n")
	helpIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "-h,") {
			helpIdx = i
			break
		}
	}
	if helpIdx <= 0 {
		return usage
	}
	out := make([]string, 0, len(lines))
	out = append(out, lines[helpIdx])
	out = append(out, lines[:helpIdx]...)
	out = append(out, lines[helpIdx+1:]...)
	return strings.Join(out, "\n")
}

// registry loads config and returns a Registry with saved credentials applied.
func registry() *api.Registry {
	cfg, _ := config.Load()
	return api.NewRegistry(api.RegistryOptions{
		NeteaseCookie:    cfg.NeteaseCookie,
		NeteaseCsrf:      cfg.NeteaseCsrf,
		NeteaseCookieRaw: cfg.NeteaseCookieRaw,
	})
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
