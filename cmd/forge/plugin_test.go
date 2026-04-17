package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectIDEFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envVal   string
		expected string
	}{
		{"cursor from CURSOR_TRACE_ID", "CURSOR_TRACE_ID", "abc", "cursor"},
		{"claude from CLAUDE_CODE", "CLAUDE_CODE", "1", "claude-code"},
		{"windsurf from CODEIUM_SESSION", "CODEIUM_SESSION", "xyz", "windsurf"},
		{"unknown fallback", "", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"CURSOR_TRACE_ID", "CURSOR_SESSION", "CLAUDE_CODE", "CLAUDE_SESSION_ID", "CODEIUM_SESSION", "WINDSURF_SESSION"} {
				t.Setenv(key, "")
			}
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := detectIDEFromEnv()
			if got != tt.expected {
				t.Errorf("detectIDEFromEnv() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateMCPConfig(t *testing.T) {
	root := t.TempDir()
	mcpDir := filepath.Join(root, "harness", "src")
	if err := os.MkdirAll(mcpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mcpDir, "mcp-server.ts"), []byte("// stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORGE_ROOT", root)

	config, err := generateMCPConfig("/usr/local/bin/npx")
	if err != nil {
		t.Fatalf("generateMCPConfig: %v", err)
	}
	if config == "" {
		t.Fatal("generateMCPConfig returned empty string")
	}
	if !strings.Contains(config, "forge") {
		t.Error("config should contain 'forge' server entry")
	}
	if !strings.Contains(config, "mcp-server") {
		t.Error("config should reference mcp-server")
	}
}

func TestWriteMCPConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	err := writeMCPConfig(configPath, `{"mcpServers":{"forge":{}}}`)
	if err != nil {
		t.Fatalf("writeMCPConfig: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "forge") {
		t.Error("written config should contain 'forge'")
	}
}

func TestFindForgeRootFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORGE_ROOT", dir)
	got := findForgeRoot()
	if got != dir {
		t.Errorf("findForgeRoot() = %q, want %q", got, dir)
	}
}

func TestGenerateMCPConfigValidatesPath(t *testing.T) {
	t.Setenv("FORGE_ROOT", t.TempDir())
	_, err := generateMCPConfig("npx")
	if err == nil {
		t.Fatal("expected error when mcp-server.ts is missing")
	}
	if !strings.Contains(err.Error(), "mcp-server.ts not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteMCPConfigMergesExisting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")

	existing := `{"mcpServers":{"other-server":{"command":"other"}}}`
	if err := os.WriteFile(configPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeMCPConfig(configPath, `{"mcpServers":{"forge":{"command":"npx"}}}`)
	if err != nil {
		t.Fatalf("writeMCPConfig merge: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "other-server") {
		t.Error("merged config should preserve existing other-server entry")
	}
	if !strings.Contains(content, "forge") {
		t.Error("merged config should contain forge entry")
	}
}

func TestWriteMCPConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")

	if err := os.WriteFile(configPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeMCPConfig(configPath, `{"mcpServers":{"forge":{}}}`)
	if err != nil {
		t.Fatalf("writeMCPConfig with invalid existing: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "forge") {
		t.Error("overwritten config should contain forge")
	}
}

func TestMCPConfigPathForIDE(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		ide        string
		wantSuffix string
	}{
		{"cursor", filepath.Join(".cursor", "mcp.json")},
		{"claude-code", ".mcp.json"},
	}

	for _, tt := range tests {
		t.Run(tt.ide, func(t *testing.T) {
			got := mcpConfigPathForIDE(tt.ide, dir)
			expected := filepath.Join(dir, tt.wantSuffix)
			if got != expected {
				t.Errorf("mcpConfigPathForIDE(%q) = %q, want %q", tt.ide, got, expected)
			}
		})
	}
}
