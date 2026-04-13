package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func forgeCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = "."
	return cmd
}

func TestCLIValidateSuccess(t *testing.T) {
	cmd := forgeCmd(t, "blueprint", "validate", "../../blueprints/standard-implementation.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "is valid") {
		t.Errorf("expected 'is valid' in output, got: %s", out)
	}
}

func TestCLIListSuccess(t *testing.T) {
	cmd := forgeCmd(t, "blueprint", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "Built-in blueprints") {
		t.Errorf("expected 'Built-in blueprints' in output, got: %s", out)
	}
}

func TestCLIValidateInvalidFile(t *testing.T) {
	cmd := forgeCmd(t, "blueprint", "validate", "/nonexistent/forge-blueprint-does-not-exist.yaml")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure, got success. output: %s", out)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got 0. output: %s", out)
	}
}

func TestCLIRunNoArgs(t *testing.T) {
	cmd := forgeCmd(t, "run")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for run with no args")
	}
	if !strings.Contains(string(out), "usage") {
		t.Errorf("expected usage message, got: %s", out)
	}
}

func TestParseBlueprintRunArgsBuiltin(t *testing.T) {
	harness, file, builtin, task, err := parseBlueprintRunArgs([]string{
		"--builtin", "standard-implementation",
		"--task", "add JSON logging",
		"--harness", "127.0.0.1:50051",
	})
	if err != nil {
		t.Fatalf("parseBlueprintRunArgs returned error: %v", err)
	}
	if harness != "127.0.0.1:50051" {
		t.Fatalf("harness = %q, want %q", harness, "127.0.0.1:50051")
	}
	if file != "" {
		t.Fatalf("file = %q, want empty", file)
	}
	if builtin != "standard-implementation" {
		t.Fatalf("builtin = %q, want %q", builtin, "standard-implementation")
	}
	if task != "add JSON logging" {
		t.Fatalf("task = %q, want %q", task, "add JSON logging")
	}
}

func TestResolveBlueprintDataBuiltinTemplate(t *testing.T) {
	data, label, err := resolveBlueprintData("", "standard-implementation", "add JSON logging")
	if err != nil {
		t.Fatalf("resolveBlueprintData returned error: %v", err)
	}
	if label != "standard-implementation" {
		t.Fatalf("label = %q, want %q", label, "standard-implementation")
	}
	got := string(data)
	if !strings.Contains(got, "add JSON logging") {
		t.Fatalf("resolved blueprint missing task substitution: %s", got)
	}
}

func TestForgeRunNoSandboxRequiresHarnessForClaude(t *testing.T) {
	cmd := forgeCmd(t, "run", "--no-sandbox", "--adapter", "claude", "ship Layer 4")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure, got success: %s", out)
	}
	if !strings.Contains(string(out), "--harness is required") {
		t.Fatalf("expected harness guidance, got: %s", out)
	}
}

func TestCLINoArgs(t *testing.T) {
	cmd := forgeCmd(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure, got success. output: %s", out)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got 0. output: %s", out)
	}
}
