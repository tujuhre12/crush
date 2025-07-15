package ollama

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// setupCleanup sets up signal handlers for cleanup
func setupCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()
}

// cleanup stops all running models and service if started by Crush
func cleanup() {
	processManager.mu.Lock()
	defer processManager.mu.Unlock()

	// Stop all running models using HTTP API
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if IsRunning(ctx) {
		stopAllModels(ctx)
	}

	// Stop Ollama service if we started it
	if processManager.crushStartedOllama && processManager.ollamaProcess != nil {
		stopOllamaService()
	}
}

// stopAllModels stops all running models
func stopAllModels(ctx context.Context) {
	runningModels, err := GetRunningModels(ctx)
	if err != nil {
		return
	}

	for _, model := range runningModels {
		stopModel(ctx, model.Name)
	}
}

// stopModel stops a specific model using CLI
func stopModel(ctx context.Context, modelName string) error {
	cmd := exec.CommandContext(ctx, "ollama", "stop", modelName)
	return cmd.Run()
}

// stopOllamaService stops the Ollama service process
func stopOllamaService() {
	if processManager.ollamaProcess == nil {
		return
	}

	// Try graceful shutdown first
	if err := processManager.ollamaProcess.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- processManager.ollamaProcess.Wait()
		}()

		select {
		case <-done:
			// Process finished gracefully
		case <-time.After(5 * time.Second):
			// Force kill if not shut down gracefully
			syscall.Kill(-processManager.ollamaProcess.Process.Pid, syscall.SIGKILL)
			processManager.ollamaProcess.Wait()
		}
	}

	processManager.ollamaProcess = nil
	processManager.crushStartedOllama = false
}
