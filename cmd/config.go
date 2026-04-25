package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/ropean/muze/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or update persistent settings",
	Long: `Set the default download directory, UI theme, and Netease cookie.

Run without flags for interactive mode (press Enter to keep current value).
Pass flags directly to update specific fields without prompts.

The Netease cookie must be the full browser cookie string containing
MUSIC_U, __csrf, JSESSIONID-WYYY and other session tokens — required
for VIP quality (FLAC / 320k) access.`,
	RunE: runConfig,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, _ := config.Load()
		_, pal := resolveTheme(cfg.Theme)
		printConfig(cfg, pal, false)
		return nil
	},
}

func init() {
	configCmd.Flags().String("dir", "", "Default download directory")
	configCmd.Flags().String("theme", "", "UI theme: base16|tech|charm|dracula|catppuccin")
	configCmd.Flags().String("cookie", "", "Full browser cookie string (MUSIC_U + __csrf + JSESSIONID etc.)")
	configCmd.AddCommand(configListCmd)
}

func runConfig(cmd *cobra.Command, _ []string) error {
	cfg, _ := config.Load()

	flagDir, _ := cmd.Flags().GetString("dir")
	flagTheme, _ := cmd.Flags().GetString("theme")
	flagCookie, _ := cmd.Flags().GetString("cookie")

	anyFlag := cmd.Flags().Changed("dir") || cmd.Flags().Changed("theme") || cmd.Flags().Changed("cookie")

	if anyFlag {
		return applyAndSave(cfg, flagDir, flagTheme, flagCookie)
	}
	return runConfigInteractive(cfg)
}

func applyAndSave(cfg *config.Config, dir, theme, cookie string) error {
	if dir != "" {
		cfg.Dir = dir
	}
	if theme != "" {
		if !isValidTheme(theme) {
			return fmt.Errorf("unknown theme %q — valid options: %s", theme, strings.Join(config.Themes, ", "))
		}
		cfg.Theme = theme
	}
	if cookie != "" {
		cfg.NeteaseCookieRaw = cookie
		cfg.NeteaseCookie = ""
		cfg.NeteaseCsrf = ""
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	_, pal := resolveTheme(cfg.Theme)
	printConfig(cfg, pal, true)
	return nil
}

func runConfigInteractive(cfg *config.Config) error {
	huhTheme, pal := resolveTheme(cfg.Theme)

	arrow := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Render("▶")
	dim := lipgloss.NewStyle().Faint(true)
	fmt.Fprintf(os.Stderr, "\n%s Configure muze  %s\n\n",
		arrow, dim.Render("(press Enter to keep current value)"))

	// --- Theme ---
	themeChoice := ""
	themeOpts := []huh.Option[string]{
		huh.NewOption("(keep current: "+cfg.Theme+")", ""),
	}
	for _, t := range config.Themes {
		themeOpts = append(themeOpts, huh.NewOption(t, t))
	}
	if err := huh.NewSelect[string]().
		Title("Theme").
		Description("UI colour scheme").
		Options(themeOpts...).
		Value(&themeChoice).
		WithTheme(huhTheme).
		Run(); err != nil {
		return err
	}

	// --- Download directory ---
	dirInput := ""
	if err := huh.NewInput().
		Title("Download directory").
		Description(fmt.Sprintf("Current: %s", currentOrDefault(cfg.Dir, config.DefaultDownloadDir()))).
		Placeholder("e.g. /Users/me/Music  (empty = keep)").
		Value(&dirInput).
		WithTheme(huhTheme).
		Run(); err != nil {
		return err
	}

	// --- Netease cookie ---
	cookieInput := ""
	cookieDesc := "Full browser cookie string containing MUSIC_U, __csrf, JSESSIONID-WYYY, etc.\n" +
		"  Required for VIP quality (FLAC / lossless). Copy from browser DevTools → Network tab."
	if cfg.NeteaseCookieRaw != "" {
		cookieDesc += "\n  Current: " + cfg.NeteaseCookieRaw[:min(60, len(cfg.NeteaseCookieRaw))] + "..."
	}
	if err := huh.NewInput().
		Title("Netease cookie").
		Description(cookieDesc).
		Placeholder("e.g. _ntes_nnid=...; MUSIC_U=...; __csrf=...  (empty = keep)").
		Value(&cookieInput).
		WithTheme(huhTheme).
		Run(); err != nil {
		return err
	}

	return applyAndSave(cfg, strings.TrimSpace(dirInput), themeChoice, strings.TrimSpace(cookieInput))
}

func printConfig(cfg *config.Config, pal Palette, saved bool) {
	arrow := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Render("▶")

	cookieSummary := "(not set)"
	if cfg.NeteaseCookieRaw != "" {
		cookieSummary = cfg.NeteaseCookieRaw[:min(40, len(cfg.NeteaseCookieRaw))] + "..."
	} else if cfg.NeteaseCookie != "" {
		cookieSummary = "MUSIC_U=" + cfg.NeteaseCookie[:min(20, len(cfg.NeteaseCookie))] + "..."
	}

	header := fmt.Sprintf("\n%s Config", arrow)
	if saved {
		header += "  " + lipgloss.NewStyle().Foreground(pal.OK).Bold(true).Render("saved")
	}
	fmt.Fprintln(os.Stderr, header)
	fmt.Fprintf(os.Stderr, "  theme   %s\n", cfg.Theme)
	fmt.Fprintf(os.Stderr, "  dir     %s\n", currentOrDefault(cfg.Dir, config.DefaultDownloadDir()))
	fmt.Fprintf(os.Stderr, "  cookie  %s\n\n", cookieSummary)
}

func isValidTheme(t string) bool {
	for _, v := range config.Themes {
		if v == t {
			return true
		}
	}
	return false
}

func currentOrDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
