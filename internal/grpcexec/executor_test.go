package grpcexec

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"testing"
	"time"

	forgev1 "github.com/aditya-soni/forge/internal/grpcexec/forgev1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// mockForgeAgentServer implements the ForgeAgent gRPC server for testing.
type mockForgeAgentServer struct {
	forgev1.UnimplementedForgeAgentServer
	lastRequest *forgev1.ExecuteAgentRequest
}

func (s *mockForgeAgentServer) ExecuteAgent(
	_ context.Context,
	req *forgev1.ExecuteAgentRequest,
) (*forgev1.ExecuteAgentResponse, error) {
	s.lastRequest = req
	return &forgev1.ExecuteAgentResponse{
		Output:  "mock output for: " + req.Prompt,
		Success: true,
	}, nil
}

func startMockServer(t *testing.T) (string, *mockForgeAgentServer, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	mock := &mockForgeAgentServer{}
	forgev1.RegisterForgeAgentServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	return lis.Addr().String(), mock, func() { srv.Stop() }
}

// failingForgeAgentServer returns Success=false (no gRPC error).
type failingForgeAgentServer struct {
	forgev1.UnimplementedForgeAgentServer
}

func (s *failingForgeAgentServer) ExecuteAgent(
	_ context.Context,
	_ *forgev1.ExecuteAgentRequest,
) (*forgev1.ExecuteAgentResponse, error) {
	return &forgev1.ExecuteAgentResponse{
		Output:  "",
		Success: false,
		Error:   "agent error from harness",
	}, nil
}

func startFailingMockServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	forgev1.RegisterForgeAgentServer(srv, &failingForgeAgentServer{})
	go func() { _ = srv.Serve(lis) }()
	return lis.Addr().String(), func() { srv.Stop() }
}

func TestGrpcAgentExecutorSuccess(t *testing.T) {
	addr, mock, cleanup := startMockServer(t)
	defer cleanup()

	executor, err := NewGrpcAgentExecutor(addr, "/tmp/project", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewGrpcAgentExecutor: %v", err)
	}
	defer executor.Close()

	config := map[string]interface{}{"model": "claude-sonnet"}
	output, err := executor.Execute(context.Background(), "Fix the bug", config)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if output != "mock output for: Fix the bug" {
		t.Errorf("output = %q, want %q", output, "mock output for: Fix the bug")
	}

	// Verify config was serialized as JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(mock.lastRequest.ConfigJson), &parsed); err != nil {
		t.Fatalf("config_json not valid JSON: %v", err)
	}
	if parsed["model"] != "claude-sonnet" {
		t.Errorf("config model = %v, want claude-sonnet", parsed["model"])
	}
	if mock.lastRequest.WorkingDirectory != "/tmp/project" {
		t.Errorf("working_directory = %q, want /tmp/project", mock.lastRequest.WorkingDirectory)
	}
}

func TestGrpcAgentExecutorServerReturnsFailure(t *testing.T) {
	addr, cleanup := startFailingMockServer(t)
	defer cleanup()

	executor, err := NewGrpcAgentExecutor(addr, "/tmp", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewGrpcAgentExecutor: %v", err)
	}
	defer executor.Close()

	_, err = executor.Execute(context.Background(), "hi", nil)
	if err == nil {
		t.Fatal("expected error when server returns success=false")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestGrpcAgentExecutorMarshalError(t *testing.T) {
	addr, _, cleanup := startMockServer(t)
	defer cleanup()

	executor, err := NewGrpcAgentExecutor(addr, "/tmp", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewGrpcAgentExecutor: %v", err)
	}
	defer executor.Close()

	_, err = executor.Execute(context.Background(), "p", map[string]interface{}{
		"bad": math.NaN(),
	})
	if err == nil {
		t.Fatal("expected marshal error for NaN in config")
	}
}

func TestGrpcAgentExecutorFailure(t *testing.T) {
	// No listener on this port — RPC should fail (bounded by context).
	executor, err := NewGrpcAgentExecutor("127.0.0.1:49151", "/tmp",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewGrpcAgentExecutor: %v", err)
	}
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = executor.Execute(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error when harness is unreachable")
	}
}
