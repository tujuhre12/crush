package ollama

import (
	"context"
	"testing"
	"time"
)

func TestStartService(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping StartService test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Check if already running
	if IsRunning(ctx) {
		t.Log("✓ Ollama is already running, skipping start test")
		return
	}

	t.Log("Starting Ollama service...")
	err := StartService(ctx)
	if err != nil {
		t.Fatalf("Failed to start Ollama service: %v", err)
	}

	// Verify it's running
	if !IsRunning(ctx) {
		t.Fatal("Ollama service was started but IsRunning returns false")
	}

	t.Log("✓ Ollama service started successfully")

	// Cleanup
	defer func() {
		if processManager.crushStartedOllama {
			cleanup()
		}
	}()
}

func TestEnsureRunning(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping EnsureRunning test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := EnsureRunning(ctx)
	if err != nil {
		t.Fatalf("EnsureRunning failed: %v", err)
	}

	if !IsRunning(ctx) {
		t.Fatal("EnsureRunning succeeded but Ollama is not running")
	}

	t.Log("✓ EnsureRunning succeeded")

	// Test calling it again when already running
	err = EnsureRunning(ctx)
	if err != nil {
		t.Fatalf("EnsureRunning failed on second call: %v", err)
	}

	t.Log("✓ EnsureRunning is idempotent")

	// Cleanup
	defer func() {
		if processManager.crushStartedOllama {
			cleanup()
		}
	}()
}

func TestEnsureModelLoaded(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping EnsureModelLoaded test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Get available models
	if err := EnsureRunning(ctx); err != nil {
		t.Fatalf("Failed to ensure Ollama is running: %v", err)
	}

	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping EnsureModelLoaded test")
	}

	// Pick a smaller model if available
	testModel := models[0].Name
	for _, model := range models {
		if model.Name == "phi3:3.8b" || model.Name == "llama3.2:3b" {
			testModel = model.Name
			break
		}
	}

	t.Logf("Testing with model: %s", testModel)

	err = EnsureModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to ensure model is loaded: %v", err)
	}

	// Verify model is loaded
	running, err := IsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running: %v", err)
	}

	if !running {
		t.Fatal("EnsureModelLoaded succeeded but model is not running")
	}

	t.Log("✓ EnsureModelLoaded succeeded")

	// Test calling it again when already loaded
	err = EnsureModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("EnsureModelLoaded failed on second call: %v", err)
	}

	t.Log("✓ EnsureModelLoaded is idempotent")

	// Cleanup
	defer func() {
		if processManager.crushStartedOllama {
			cleanup()
		}
	}()
}
