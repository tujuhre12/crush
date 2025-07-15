package ollama

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

var processManager = &ProcessManager{}

// StartService starts the Ollama service if not already running
func StartService(ctx context.Context) error {
	if IsRunning(ctx) {
		return nil // Already running
	}

	if !IsInstalled() {
		return fmt.Errorf("Ollama is not installed")
	}

	processManager.mu.Lock()
	defer processManager.mu.Unlock()

	// Set up cleanup on first use
	processManager.setupOnce.Do(func() {
		setupCleanup()
	})

	// Start Ollama service
	cmd := exec.CommandContext(ctx, "ollama", "serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama service: %w", err)
	}

	processManager.ollamaProcess = cmd
	processManager.crushStartedOllama = true

	// Wait for service to be ready
	startTime := time.Now()
	for time.Since(startTime) < ServiceStartTimeout {
		if IsRunning(ctx) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("Ollama service did not start within %v", ServiceStartTimeout)
}

// EnsureRunning ensures Ollama service is running, starting it if necessary
func EnsureRunning(ctx context.Context) error {
	// Always ensure cleanup is set up, even if Ollama was already running
	processManager.setupOnce.Do(func() {
		setupCleanup()
	})
	return StartService(ctx)
}

// EnsureModelLoaded ensures a model is loaded, loading it if necessary
func EnsureModelLoaded(ctx context.Context, modelName string) error {
	if err := EnsureRunning(ctx); err != nil {
		return err
	}

	running, err := IsModelRunning(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to check if model is running: %w", err)
	}

	if running {
		return nil // Already loaded
	}

	// Load the model
	loadCtx, cancel := context.WithTimeout(ctx, ModelLoadTimeout)
	defer cancel()

	if err := LoadModel(loadCtx, modelName); err != nil {
		return fmt.Errorf("failed to load model %s: %w", modelName, err)
	}

	return nil
}
