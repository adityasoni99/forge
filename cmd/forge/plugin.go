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

	if ide == "auto" {
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

	config := generateMCPConfig(npxPath)
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
	return "auto"
}

func mcpConfigPathForIDE(ide, projectDir string) string {
	switch ide {
	case "cursor":
		return filepath.Join(projectDir, ".cursor", "mcp.json")
	case "claude-code":
		return filepath.Join(projectDir, ".mcp.json")
	case "windsurf":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	default:
		return filepath.Join(projectDir, ".cursor", "mcp.json")
	}
}

func generateMCPConfig(npxPath string) string {
	type serverEntry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	type mcpConfig struct {
		MCPServers map[string]serverEntry `json:"mcpServers"`
	}

	config := mcpConfig{
		MCPServers: map[string]serverEntry{
			"forge": {
				Command: npxPath,
				Args:    []string{"tsx", "src/mcp-server.ts"},
			},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func writeMCPConfig(path, content string) error {
	existing, err := os.ReadFile(path)
	if err == nil {
		var existingConfig map[string]interface{}
		if json.Unmarshal(existing, &existingConfig) == nil {
			var newConfig map[string]interface{}
			if json.Unmarshal([]byte(content), &newConfig) == nil {
				if servers, ok := existingConfig["mcpServers"].(map[string]interface{}); ok {
					if newServers, ok := newConfig["mcpServers"].(map[string]interface{}); ok {
						for k, v := range newServers {
							servers[k] = v
						}
						existingConfig["mcpServers"] = servers
						merged, _ := json.MarshalIndent(existingConfig, "", "  ")
						return os.WriteFile(path, merged, 0o644)
					}
				}
			}
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
