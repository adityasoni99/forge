package sandbox

import "fmt"

type SandboxError struct {
	ExitCode int
	Output   string
	Cause    error
}

func (e *SandboxError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("sandbox: container exited %d: %v", e.ExitCode, e.Cause)
	}
	return fmt.Sprintf("sandbox: container exited %d", e.ExitCode)
}

func (e *SandboxError) Unwrap() error {
	return e.Cause
}
