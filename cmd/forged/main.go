package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aditya-soni/forge/factory/delivery"
	"github.com/aditya-soni/forge/factory/orchestrator"
	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/triggers"
	"github.com/aditya-soni/forge/factory/workspace"
)

func main() {
	port := flag.String("port", envOr("FORGED_PORT", "8080"), "HTTP listen port")
	maxParallel := flag.Int("max-parallel", envOrInt("FORGED_MAX_PARALLEL", 2), "max concurrent runs")
	dryRun := flag.Bool("dry-run", false, "use log-only pipeline (no Docker/git)")
	sessionsDir := flag.String("sessions-dir", envOr("FORGED_SESSIONS_DIR", ".forge/sessions"), "session log directory")
	repoCacheDir := flag.String("repo-cache-dir", envOr("FORGED_REPO_CACHE", ".forge/repo-cache"), "bare clone cache for repo_url resolution")
	warmPoolSize := flag.Int("warm-pool-size", envOrInt("FORGED_WARM_POOL_SIZE", 0), "warm container pool size (0=disabled)")
	warmPoolImage := flag.String("warm-pool-image", envOr("FORGED_WARM_POOL_IMAGE", "forge:latest"), "image for warm pool containers")
	lazySandboxFlag := flag.Bool("lazy-sandbox", true, "defer sandbox provisioning until first sandbox-bound node")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registry := orchestrator.NewRunRegistry()
	sessionLog := orchestrator.NewFileSessionLog(*sessionsDir)

	var pipeline orchestrator.PipelineExecutor
	var warmPool *sandbox.DockerWarmPool
	if *dryRun {
		pipeline = &logPipeline{}
	} else {
		sbx := sandbox.NewDockerSandbox(nil)
		if *warmPoolSize > 0 {
			warmPool = sandbox.NewDockerWarmPool(sandbox.NewExecRunner(), *warmPoolSize)
			warmPool.Preheat(ctx, sandbox.SandboxConfig{Image: *warmPoolImage})
			sbx.SetWarmPool(warmPool)
		}
		ws := workspace.NewManager()
		dlv := delivery.NewGitDelivery(&sandbox.ExecRunner{})
		assigner := orchestrator.NewTaskAssigner()

		pipelineOpts := []orchestrator.PipelineOption{
			orchestrator.WithTaskAssigner(assigner),
			orchestrator.WithSessionLog(sessionLog),
		}
		if *lazySandboxFlag {
			pipelineOpts = append(pipelineOpts, orchestrator.WithLazySandbox(true))
		}
		pipeline = orchestrator.NewPipeline(sbx, ws, dlv, pipelineOpts...)
	}
	queue := orchestrator.NewRunQueue(registry, pipeline, *maxParallel)
	go queue.Start(ctx)

	resolver := triggers.NewGitRepoResolver(*repoCacheDir)
	handler := triggers.NewWebhookHandler(queue, registry, triggers.WithRepoResolver(resolver))
	mux := http.NewServeMux()
	mux.Handle("/api/v1/runs", handler)
	mux.Handle("/api/v1/runs/", handler)

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("forged listening on :%s (max_parallel=%d, dry_run=%v)", *port, *maxParallel, *dryRun)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := queue.Shutdown(shutdownCtx); err != nil {
		log.Printf("queue shutdown: %v", err)
	}
	if warmPool != nil {
		if err := warmPool.Shutdown(shutdownCtx); err != nil {
			log.Printf("warm pool shutdown: %v", err)
		}
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	log.Println("shutdown complete")
}

type logPipeline struct{}

func (p *logPipeline) Execute(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunResult, error) {
	log.Printf("dry-run: task=%q blueprint=%q adapter=%q", req.Task, req.BlueprintName, req.Adapter)
	return orchestrator.RunResult{
		Status: orchestrator.RunStatusPassed,
		Output: fmt.Sprintf("dry-run: %s", req.Task),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		log.Printf("warning: invalid %s=%q, using default %d", key, v, fallback)
		return fallback
	}
	return n
}
