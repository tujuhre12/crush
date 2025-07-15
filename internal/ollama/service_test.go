package ollama

import (
	"context"
	"testing"
	"time"
)

func TestStartOllamaService(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping StartOllamaService test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// First check if it's already running
	if IsRunning(ctx) {
		t.Log("Ollama is already running, skipping start test")
		return
	}

	t.Log("Starting Ollama service...")
	err := StartOllamaService(ctx)
	if err != nil {
		t.Fatalf("Failed to start Ollama service: %v", err)
	}

	// Verify it's now running
	if !IsRunning(ctx) {
		t.Fatal("Ollama service was started but IsRunning still returns false")
	}

	t.Log("Ollama service started successfully")

	// Clean up - stop the service we started
	cleanupProcesses()
}

func TestEnsureOllamaRunning(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping EnsureOllamaRunning test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test that EnsureOllamaRunning works whether Ollama is running or not
	err := EnsureOllamaRunning(ctx)
	if err != nil {
		t.Fatalf("EnsureOllamaRunning failed: %v", err)
	}

	// Verify Ollama is running
	if !IsRunning(ctx) {
		t.Fatal("EnsureOllamaRunning succeeded but Ollama is not running")
	}

	t.Log("EnsureOllamaRunning succeeded")

	// Test calling it again when already running
	err = EnsureOllamaRunning(ctx)
	if err != nil {
		t.Fatalf("EnsureOllamaRunning failed on second call: %v", err)
	}

	t.Log("EnsureOllamaRunning works when already running")
}

func TestStartModel(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping StartModel test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Starting Ollama service...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	// Get available models
	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping StartModel test")
	}

	// Pick a smaller model if available, otherwise use the first one
	testModel := models[0].ID
	for _, model := range models {
		if model.ID == "phi3:3.8b" || model.ID == "llama3.2:3b" {
			testModel = model.ID
			break
		}
	}

	t.Logf("Testing with model: %s", testModel)

	// Check if model is already loaded
	loaded, err := IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded: %v", err)
	}

	if loaded {
		t.Log("Model is already loaded, skipping start test")
		return
	}

	t.Log("Starting model...")
	err = StartModel(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to start model: %v", err)
	}

	// Verify model is now loaded
	loaded, err = IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded after start: %v", err)
	}

	if !loaded {
		t.Fatal("StartModel succeeded but model is not loaded")
	}

	t.Log("Model started successfully")
}

func TestEnsureModelRunning(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping EnsureModelRunning test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Starting Ollama service...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	// Get available models
	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping EnsureModelRunning test")
	}

	testModel := models[0].ID
	t.Logf("Testing with model: %s", testModel)

	// Test EnsureModelRunning
	err = EnsureModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("EnsureModelRunning failed: %v", err)
	}

	// Verify model is running
	loaded, err := IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded: %v", err)
	}

	if !loaded {
		t.Fatal("EnsureModelRunning succeeded but model is not loaded")
	}

	t.Log("EnsureModelRunning succeeded")

	// Test calling it again when already running
	err = EnsureModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("EnsureModelRunning failed on second call: %v", err)
	}

	t.Log("EnsureModelRunning works when model already running")
}
