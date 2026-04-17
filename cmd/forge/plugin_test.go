package main

import (
	"os"
	"path/filepath"
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
		{"auto fallback", "", "", "auto"},
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
	config := generateMCPConfig("/usr/local/bin/npx")
	if config == "" {
		t.Fatal("generateMCPConfig returned empty string")
	}
	if !containsStr(config, "forge") {
		t.Error("config should contain 'forge' server entry")
	}
	if !containsStr(config, "mcp-server") {
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
	if !containsStr(string(data), "forge") {
		t.Error("written config should contain 'forge'")
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

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
