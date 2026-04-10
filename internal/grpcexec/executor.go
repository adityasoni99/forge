package grpcexec

import (
	"context"
	"encoding/json"
	"fmt"

	forgev1 "github.com/aditya-soni/forge/internal/grpcexec/forgev1"
	"google.golang.org/grpc"
)

// GrpcAgentExecutor implements blueprint.AgentExecutor by calling the
// TypeScript harness over gRPC.
type GrpcAgentExecutor struct {
	client     forgev1.ForgeAgentClient
	conn       *grpc.ClientConn
	workingDir string
}

// NewGrpcAgentExecutor dials the harness at addr and returns an executor rooted at workingDir.
func NewGrpcAgentExecutor(addr, workingDir string, opts ...grpc.DialOption) (*GrpcAgentExecutor, error) {
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", addr, err)
	}
	return &GrpcAgentExecutor{
		client:     forgev1.NewForgeAgentClient(conn),
		conn:       conn,
		workingDir: workingDir,
	}, nil
}

// Close releases the gRPC connection.
func (e *GrpcAgentExecutor) Close() error {
	return e.conn.Close()
}

// Execute calls the harness ExecuteAgent RPC with JSON-serialized config.
func (e *GrpcAgentExecutor) Execute(ctx context.Context, prompt string, config map[string]interface{}) (string, error) {
	configJSON := "{}"
	if config != nil {
		data, err := json.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("marshal config: %w", err)
		}
		configJSON = string(data)
	}

	resp, err := e.client.ExecuteAgent(ctx, &forgev1.ExecuteAgentRequest{
		Prompt:           prompt,
		ConfigJson:       configJSON,
		WorkingDirectory: e.workingDir,
	})
	if err != nil {
		return "", fmt.Errorf("ExecuteAgent RPC: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("agent error: %s", resp.Error)
	}
	return resp.Output, nil
}
