package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdPlugin(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: forge plugin <install|status> [flags]")
		os.Exit(1)
	}

	switch args[0] {
	case "install":
		cmdPluginInstall(args[1:])
	case "status":
		cmdPluginStatus()
	default:
		fmt.Fprintf(os.Stderr, "unknown plugin subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdPluginInstall(args []string) {
	ide := "auto"
	projectDir := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ide":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--ide requires a value (cursor, claude-code, windsurf, auto)")
				os.Exit(1)
			}
			i++
			ide = args[i]
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a path")
				os.Exit(1)
			}
			i++
			projectDir = args[i]
		}
	}

	if ide == "auto" || ide == "unknown" {
		ide = detectIDEFromEnv()
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve dir: %v\n", err)
		os.Exit(1)
	}

	npxPath, err := exec.LookPath("npx")
	if err != nil {
		npxPath = "npx"
	}

	config, err := generateMCPConfig(npxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate config: %v\n", err)
		os.Exit(1)
	}
	configPath := mcpConfigPathForIDE(ide, absDir)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create dir: %v\n", err)
		os.Exit(1)
	}

	if err := writeMCPConfig(configPath, config); err != nil {
		fmt.Fprintf(os.Stderr, "write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Forge plugin installed for %s\n", ide)
	fmt.Printf("Config written to: %s\n", configPath)
	fmt.Println("Restart your IDE to activate the Forge MCP tools.")
}

func cmdPluginStatus() {
	ide := detectIDEFromEnv()
	fmt.Printf("Detected IDE: %s\n", ide)

	cwd, _ := os.Getwd()
	configPath := mcpConfigPathForIDE(ide, cwd)
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config found: %s\n", configPath)
		fmt.Println("Status: installed")
	} else {
		fmt.Println("Status: not installed")
		fmt.Println("Run 'forge plugin install' to set up the MCP plugin.")
	}
}

func detectIDEFromEnv() string {
	if os.Getenv("CURSOR_TRACE_ID") != "" || os.Getenv("CURSOR_SESSION") != "" {
		return "cursor"
	}
	if os.Getenv("CLAUDE_CODE") != "" || os.Getenv("CLAUDE_SESSION_ID") != "" {
		return "claude-code"
	}
	if os.Getenv("CODEIUM_SESSION") != "" || os.Getenv("WINDSURF_SESSION") != "" {
		return "windsurf"
	}
	return "unknown"
}

func mcpConfigPathForIDE(ide, projectDir string) string {
	switch ide {
	case "cursor":
		return filepath.Join(projectDir, ".cursor", "mcp.json")
	case "claude-code":
		return filepath.Join(projectDir, ".mcp.json")
	case "windsurf":
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not determine home directory, using %s\n", os.TempDir())
			home = os.TempDir()
		}
		return filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	default:
		return filepath.Join(projectDir, ".cursor", "mcp.json")
	}
}

func generateMCPConfig(npxPath string) (string, error) {
	type serverEntry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	type mcpConfig struct {
		MCPServers map[string]serverEntry `json:"mcpServers"`
	}

	forgeRoot := findForgeRoot()
	mcpServerPath := filepath.Join(forgeRoot, "harness", "src", "mcp-server.ts")
	if _, err := os.Stat(mcpServerPath); err != nil {
		return "", fmt.Errorf("mcp-server.ts not found at %s (set FORGE_ROOT to override)", mcpServerPath)
	}

	config := mcpConfig{
		MCPServers: map[string]serverEntry{
			"forge": {
				Command: npxPath,
				Args:    []string{"tsx", mcpServerPath, "--mcp"},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	return string(data), nil
}

func findForgeRoot() string {
	if env := os.Getenv("FORGE_ROOT"); env != "" {
		return env
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	for _, candidate := range []string{
		dir,
		filepath.Dir(dir),
		filepath.Dir(filepath.Dir(dir)),
	} {
		if _, err := os.Stat(filepath.Join(candidate, "harness", "src", "mcp-server.ts")); err == nil {
			return candidate
		}
	}
	return dir
}

func writeMCPConfig(path, content string) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		return os.WriteFile(path, []byte(content), 0o644)
	}

	var existingConfig map[string]interface{}
	if err := json.Unmarshal(existing, &existingConfig); err != nil {
		fmt.Fprintf(os.Stderr, "warning: existing %s is not valid JSON, overwriting\n", path)
		return os.WriteFile(path, []byte(content), 0o644)
	}

	var newConfig map[string]interface{}
	if err := json.Unmarshal([]byte(content), &newConfig); err != nil {
		return fmt.Errorf("marshal new config: %w", err)
	}

	existingServers, ok := existingConfig["mcpServers"].(map[string]interface{})
	if !ok {
		if existingConfig["mcpServers"] != nil {
			fmt.Fprintf(os.Stderr, "warning: existing %s has unexpected mcpServers format, overwriting\n", path)
		}
		existingConfig["mcpServers"] = newConfig["mcpServers"]
	} else if newServers, ok := newConfig["mcpServers"].(map[string]interface{}); ok {
		for k, v := range newServers {
			existingServers[k] = v
		}
		existingConfig["mcpServers"] = existingServers
	}

	merged, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal merged config: %w", err)
	}
	return os.WriteFile(path, merged, 0o644)
}
