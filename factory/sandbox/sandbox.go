// factory/sandbox/sandbox.go
package sandbox

import "context"

type SandboxManager interface {
	EnsureImage(ctx context.Context, config SandboxConfig) error
	Run(ctx context.Context, config SandboxConfig, command []string) (SandboxResult, error)
}
