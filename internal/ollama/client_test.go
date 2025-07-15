package ollama

import (
	"context"
	"testing"
	"time"
)

func TestIsRunning(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping IsRunning test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	running := IsRunning(ctx)

	if running {
		t.Log("✓ Ollama is running")
	} else {
		t.Log("✗ Ollama is not running")
	}
}

func TestGetModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetModels test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsRunning(ctx) {
		t.Skip("Ollama is not running, skipping GetModels test")
	}

	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	t.Logf("✓ Found %d models", len(models))
	for _, model := range models {
		t.Logf("  - %s (size: %d bytes)", model.Name, model.Size)
	}
}

func TestGetRunningModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetRunningModels test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsRunning(ctx) {
		t.Skip("Ollama is not running, skipping GetRunningModels test")
	}

	runningModels, err := GetRunningModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get running models: %v", err)
	}

	t.Logf("✓ Found %d running models", len(runningModels))
	for _, model := range runningModels {
		t.Logf("  - %s (size: %d bytes)", model.Name, model.Size)
	}
}

func TestIsModelRunning(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping IsModelRunning test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsRunning(ctx) {
		t.Skip("Ollama is not running, skipping IsModelRunning test")
	}

	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping IsModelRunning test")
	}

	testModel := models[0].Name
	running, err := IsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running: %v", err)
	}

	if running {
		t.Logf("✓ Model %s is running", testModel)
	} else {
		t.Logf("✗ Model %s is not running", testModel)
	}
}

func BenchmarkIsRunning(b *testing.B) {
	if !IsInstalled() {
		b.Skip("Ollama is not installed")
	}

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		IsRunning(ctx)
	}
}

func BenchmarkGetModels(b *testing.B) {
	if !IsInstalled() {
		b.Skip("Ollama is not installed")
	}

	ctx := context.Background()

	if !IsRunning(ctx) {
		b.Skip("Ollama is not running")
	}

	for i := 0; i < b.N; i++ {
		GetModels(ctx)
	}
}
