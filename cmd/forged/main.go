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

	"github.com/aditya-soni/forge/factory/orchestrator"
	"github.com/aditya-soni/forge/factory/triggers"
)

func main() {
	port := flag.String("port", envOr("FORGED_PORT", "8080"), "HTTP listen port")
	maxParallel := flag.Int("max-parallel", envOrInt("FORGED_MAX_PARALLEL", 2), "max concurrent runs")
	flag.Parse()

	// Known limitation: On shutdown, in-flight queue workers complete using the
	// same context that stops the queue loop. Buffered but unstarted items may
	// be lost. A drain/wait API on RunQueue is planned for a future iteration.

	registry := orchestrator.NewRunRegistry()

	// In production, this would be a real Pipeline. For the daemon skeleton,
	// we use a placeholder that logs and returns success.
	pipeline := &logPipeline{}
	queue := orchestrator.NewRunQueue(registry, pipeline, *maxParallel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.Start(ctx)

	handler := triggers.NewWebhookHandler(queue, registry)
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
		log.Printf("forged listening on :%s (max_parallel=%d)", *port, *maxParallel)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type logPipeline struct{}

func (p *logPipeline) Execute(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunResult, error) {
	log.Printf("executing: task=%q blueprint=%q adapter=%q", req.Task, req.BlueprintName, req.Adapter)
	return orchestrator.RunResult{
		Status: orchestrator.RunStatusPassed,
		Output: fmt.Sprintf("executed task: %s", req.Task),
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
