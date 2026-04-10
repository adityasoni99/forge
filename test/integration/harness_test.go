package integration

import (
	"context"
	"net"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/aditya-soni/forge/core/blueprint"
	"github.com/aditya-soni/forge/internal/grpcexec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const harnessAddr = "127.0.0.1:50052"

func waitForTCP(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for TCP listener on %s", addr)
}

func TestEngineWithHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	harnessCmd := exec.Command("npx", "tsx", "src/server.ts")
	harnessCmd.Dir = "../../harness"
	harnessCmd.Env = append(harnessCmd.Environ(),
		"FORGE_ADAPTER=echo",
		"FORGE_HARNESS_PORT=50052",
	)
	if runtime.GOOS != "windows" {
		harnessCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if err := harnessCmd.Start(); err != nil {
		t.Fatalf("start harness: %v", err)
	}
	defer func() {
		if harnessCmd.Process == nil {
			return
		}
		if runtime.GOOS != "windows" {
			_ = syscall.Kill(-harnessCmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = harnessCmd.Process.Kill()
		}
		_ = harnessCmd.Wait()
	}()

	waitForTCP(t, harnessAddr, 30*time.Second)

	executor, err := grpcexec.NewGrpcAgentExecutor(
		harnessAddr,
		t.TempDir(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewGrpcAgentExecutor: %v", err)
	}
	defer func() { _ = executor.Close() }()

	g := blueprint.NewGraph()
	_ = g.AddNode(blueprint.NewAgenticNode("plan", "hello-harness", map[string]interface{}{"e2e": true}, executor))
	_ = g.AddNode(blueprint.NewDeterministicNode("commit", "echo e2e-ok"))
	_ = g.AddEdge(blueprint.Edge{From: "plan", To: "commit"})
	if err := g.SetStartNode("plan"); err != nil {
		t.Fatalf("SetStartNode: %v", err)
	}

	engine := blueprint.NewEngine(g, "e2e-harness")
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if state.Status != blueprint.NodeStatusPassed {
		t.Fatalf("Status = %v, want Passed", state.Status)
	}
	if len(state.NodeResults) != 2 {
		t.Fatalf("node results count = %d, want 2", len(state.NodeResults))
	}
	plan, ok := state.NodeResults["plan"]
	if !ok {
		t.Fatal("missing result for plan")
	}
	if plan.Status != blueprint.NodeStatusPassed {
		t.Errorf("plan status = %v, want Passed", plan.Status)
	}
	if plan.Output == "" {
		t.Error("expected non-empty agent output from echo harness")
	}
	commit, ok := state.NodeResults["commit"]
	if !ok {
		t.Fatal("missing result for commit")
	}
	if commit.Status != blueprint.NodeStatusPassed {
		t.Errorf("commit status = %v, want Passed", commit.Status)
	}
	if commit.Output != "e2e-ok" {
		t.Errorf("commit output = %q, want %q", commit.Output, "e2e-ok")
	}
}
