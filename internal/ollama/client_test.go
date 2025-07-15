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
		t.Log("Ollama is running")
	} else {
		t.Log("Ollama is not running")
	}

	// This test doesn't fail - it's informational
	// The behavior depends on whether Ollama is actually running
}

func TestGetModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetModels test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Ollama is not running, attempting to start...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	t.Logf("Found %d models:", len(models))
	for _, model := range models {
		t.Logf("  - %s (context: %d, max_tokens: %d)",
			model.ID, model.ContextWindow, model.DefaultMaxTokens)
	}
}

func TestGetRunningModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetRunningModels test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Ollama is not running, attempting to start...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	runningModels, err := GetRunningModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get running models: %v", err)
	}

	t.Logf("Found %d running models:", len(runningModels))
	for _, model := range runningModels {
		t.Logf("  - %s", model.Name)
	}
}

func TestIsModelLoaded(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping IsModelLoaded test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Ollama is not running, attempting to start...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	// Get available models first
	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping IsModelLoaded test")
	}

	testModel := models[0].ID
	t.Logf("Testing model: %s", testModel)

	loaded, err := IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded: %v", err)
	}

	if loaded {
		t.Logf("Model %s is loaded", testModel)
	} else {
		t.Logf("Model %s is not loaded", testModel)
	}
}

func TestGetProvider(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetProvider test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Ollama is not running, attempting to start...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	provider, err := GetProvider(ctx)
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if provider.Name != "Ollama" {
		t.Errorf("Expected provider name to be 'Ollama', got '%s'", provider.Name)
	}

	if provider.ID != "ollama" {
		t.Errorf("Expected provider ID to be 'ollama', got '%s'", provider.ID)
	}

	t.Logf("Provider: %s (ID: %s) with %d models",
		provider.Name, provider.ID, len(provider.Models))
}

func TestGetContextWindow(t *testing.T) {
	tests := []struct {
		family   string
		expected int64
	}{
		{"llama", 131072},
		{"mistral", 32768},
		{"gemma", 8192},
		{"qwen", 131072},
		{"qwen2", 131072},
		{"phi", 131072},
		{"codellama", 16384},
		{"unknown", 8192},
	}

	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			result := getContextWindow(tt.family)
			if result != tt.expected {
				t.Errorf("getContextWindow(%s) = %d, expected %d",
					tt.family, result, tt.expected)
			}
		})
	}
}

func TestSupportsImages(t *testing.T) {
	tests := []struct {
		family   string
		expected bool
	}{
		{"llama-vision", true},
		{"llava", true},
		{"llama", false},
		{"mistral", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			result := supportsImages(tt.family)
			if result != tt.expected {
				t.Errorf("supportsImages(%s) = %v, expected %v",
					tt.family, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for client functions
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

	// Ensure Ollama is running for benchmark
	if !IsRunning(ctx) {
		b.Skip("Ollama is not running")
	}

	for i := 0; i < b.N; i++ {
		GetModels(ctx)
	}
}
