package cmd

import (
	"testing"
)

func TestRootHasAllSubcommands(t *testing.T) {
	want := []string{"search", "url", "download", "serve", "version", "check-update", "upgrade"}
	commands := root.Commands()
	names := make(map[string]bool, len(commands))
	for _, c := range commands {
		names[c.Name()] = true
	}
	for _, name := range want {
		if !names[name] {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

func TestSearchCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"page":    "int",
		"limit":   "int",
		"sources": "string",
	}
	for name, typ := range flags {
		f := searchCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("search command missing flag --%s", name)
			continue
		}
		if f.Value.Type() != typ {
			t.Errorf("search --%s: expected type %s, got %s", name, typ, f.Value.Type())
		}
	}
}

func TestSearchCmd_RequiresArgs(t *testing.T) {
	if searchCmd.Args == nil {
		t.Fatal("search command should require args")
	}
	if err := searchCmd.Args(searchCmd, []string{}); err == nil {
		t.Error("search command should reject zero args")
	}
	if err := searchCmd.Args(searchCmd, []string{"keyword"}); err != nil {
		t.Errorf("search command should accept one arg: %v", err)
	}
}

func TestURLCmd_RequiresArgs(t *testing.T) {
	if urlCmd.Args == nil {
		t.Fatal("url command should require args")
	}
	if err := urlCmd.Args(urlCmd, []string{}); err == nil {
		t.Error("url command should reject zero args")
	}
	if err := urlCmd.Args(urlCmd, []string{"netease"}); err == nil {
		t.Error("url command should reject one arg (needs two)")
	}
	if err := urlCmd.Args(urlCmd, []string{"netease", "123"}); err != nil {
		t.Errorf("url command should accept two args: %v", err)
	}
}

func TestDownloadCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"out":    "string",
		"title":  "string",
		"artist": "string",
		"force":  "bool",
	}
	for name, typ := range flags {
		f := downloadCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("download command missing flag --%s", name)
			continue
		}
		if f.Value.Type() != typ {
			t.Errorf("download --%s: expected type %s, got %s", name, typ, f.Value.Type())
		}
	}
}

func TestDownloadCmd_RequiresArgs(t *testing.T) {
	if downloadCmd.Args == nil {
		t.Fatal("download command should require args")
	}
	if err := downloadCmd.Args(downloadCmd, []string{}); err == nil {
		t.Error("download command should reject zero args")
	}
	if err := downloadCmd.Args(downloadCmd, []string{"netease"}); err == nil {
		t.Error("download command should reject one arg (needs two)")
	}
	if err := downloadCmd.Args(downloadCmd, []string{"netease", "123"}); err != nil {
		t.Errorf("download command should accept two args: %v", err)
	}
}

func TestServeCmd_Flags(t *testing.T) {
	f := serveCmd.Flags().Lookup("port")
	if f == nil {
		t.Fatal("serve command missing flag --port")
	}
	if f.Value.Type() != "int" {
		t.Errorf("serve --port: expected type int, got %s", f.Value.Type())
	}
	if f.DefValue != "8010" {
		t.Errorf("serve --port default: expected 8010, got %s", f.DefValue)
	}
}

func TestUpgradeCmd_Flags(t *testing.T) {
	f := upgradeCmd.Flags().Lookup("version")
	if f == nil {
		t.Fatal("upgrade command missing flag --version")
	}
	if f.DefValue != "latest" {
		t.Errorf("upgrade --version default: expected 'latest', got %s", f.DefValue)
	}
}

func TestSearchCmd_FlagDefaults(t *testing.T) {
	f := searchCmd.Flags().Lookup("page")
	if f.DefValue != "1" {
		t.Errorf("search --page default: expected 1, got %s", f.DefValue)
	}
	f = searchCmd.Flags().Lookup("limit")
	if f.DefValue != "30" {
		t.Errorf("search --limit default: expected 30, got %s", f.DefValue)
	}
}
