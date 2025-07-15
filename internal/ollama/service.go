package ollama

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// StartOllamaService starts the Ollama service if it's not already running
func StartOllamaService(ctx context.Context) error {
	if IsRunning(ctx) {
		return nil // Already running
	}

	// Set up signal handling for cleanup
	processManager.setupOnce.Do(func() {
		setupProcessCleanup()
	})

	// Start ollama serve
	cmd := exec.CommandContext(ctx, "ollama", "serve")
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil // Suppress errors
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group so we can kill it and all children
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama service: %w", err)
	}

	// Store the process for cleanup
	processManager.mu.Lock()
	processManager.ollamaServer = cmd
	processManager.crushStartedOllama = true
	processManager.mu.Unlock()

	// Wait for Ollama to be ready (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for Ollama service to start")
		case <-ticker.C:
			if IsRunning(ctx) {
				return nil // Ollama is now running
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// StartModel starts a model using `ollama run` and keeps it loaded
func StartModel(ctx context.Context, modelName string) error {
	// Check if model is already running
	if loaded, err := IsModelLoaded(ctx, modelName); err != nil {
		return fmt.Errorf("failed to check if model is loaded: %w", err)
	} else if loaded {
		return nil // Model is already running
	}

	// Set up signal handling for cleanup
	processManager.setupOnce.Do(func() {
		setupProcessCleanup()
	})

	// Start the model in the background
	cmd := exec.CommandContext(ctx, "ollama", "run", modelName)
	cmd.Stdin = nil  // No interactive input
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil // Suppress errors

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start model %s: %w", modelName, err)
	}

	// Store the process for cleanup
	processManager.mu.Lock()
	processManager.processes[modelName] = cmd
	processManager.mu.Unlock()

	// Wait for the model to be loaded (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for model %s to load", modelName)
		case <-ticker.C:
			if loaded, err := IsModelLoaded(ctx, modelName); err != nil {
				return fmt.Errorf("failed to check if model is loaded: %w", err)
			} else if loaded {
				return nil // Model is now running
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// EnsureOllamaRunning ensures Ollama service is running, starting it if necessary
func EnsureOllamaRunning(ctx context.Context) error {
	return StartOllamaService(ctx)
}

// EnsureModelRunning ensures a model is running, starting it if necessary
func EnsureModelRunning(ctx context.Context, modelName string) error {
	return StartModel(ctx, modelName)
}
