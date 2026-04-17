package sandbox

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSandboxErrorMessage(t *testing.T) {
	err := &SandboxError{ExitCode: 137, Output: "Killed", Cause: fmt.Errorf("OOM")}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
	if !strings.Contains(msg, "137") {
		t.Errorf("message %q should contain exit code 137", msg)
	}

	errNoCause := &SandboxError{ExitCode: 1}
	if !strings.Contains(errNoCause.Error(), "1") {
		t.Errorf("message without cause %q should contain exit code", errNoCause.Error())
	}
}

func TestSandboxErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("OOM killed")
	err := &SandboxError{ExitCode: 137, Cause: cause}
	if !errors.Is(err, cause) {
		t.Error("Unwrap should return cause")
	}
}

func TestIsSandboxError(t *testing.T) {
	err := &SandboxError{ExitCode: 1}
	var target *SandboxError
	if !errors.As(err, &target) {
		t.Error("errors.As should match SandboxError")
	}
	if target.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", target.ExitCode)
	}
}
