package ollama

import (
	"context"
	"testing"
	"time"
)

func TestCLIListModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping CLI test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := CLIListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models via CLI: %v", err)
	}

	t.Logf("Found %d models via CLI", len(models))
	for _, model := range models {
		t.Logf("  - %s", model.Name)
	}
}

func TestCLIListRunningModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping CLI test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Starting Ollama service...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	runningModels, err := CLIListRunningModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list running models via CLI: %v", err)
	}

	t.Logf("Found %d running models via CLI", len(runningModels))
	for _, model := range runningModels {
		t.Logf("  - %s", model)
	}
}

func TestCLIStopAllModels(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping CLI test")
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
		t.Skip("No models available, skipping CLI stop test")
	}

	// Pick a small model for testing
	testModel := models[0].ID
	for _, model := range models {
		if model.ID == "phi3:3.8b" || model.ID == "llama3.2:3b" {
			testModel = model.ID
			break
		}
	}

	t.Logf("Testing CLI stop with model: %s", testModel)

	// Check if model is running
	running, err := CLIIsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running: %v", err)
	}

	// If not running, start it
	if !running {
		t.Log("Starting model for CLI stop test...")
		if err := StartModel(ctx, testModel); err != nil {
			t.Fatalf("Failed to start model: %v", err)
		}

		// Verify it's now running
		running, err = CLIIsModelRunning(ctx, testModel)
		if err != nil {
			t.Fatalf("Failed to check if model is running after start: %v", err)
		}
		if !running {
			t.Fatal("Model failed to start")
		}
		t.Log("Model started successfully")
	} else {
		t.Log("Model was already running")
	}

	// Now test CLI stop
	t.Log("Testing CLI stop all models...")
	if err := CLIStopAllModels(ctx); err != nil {
		t.Fatalf("Failed to stop all models via CLI: %v", err)
	}

	// Give some time for models to stop
	time.Sleep(2 * time.Second)

	// Check if model is still running
	running, err = CLIIsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running after stop: %v", err)
	}

	if running {
		t.Error("Model is still running after CLI stop")
	} else {
		t.Log("Model successfully stopped via CLI")
	}
}

func TestCLIvsHTTPPerformance(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping performance test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if !IsRunning(ctx) {
		t.Log("Starting Ollama service...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}
		defer cleanupProcesses()
	}

	results, err := BenchmarkCLIvsHTTP(ctx)
	if err != nil {
		t.Fatalf("Failed to benchmark CLI vs HTTP: %v", err)
	}

	t.Log("Performance Comparison (CLI vs HTTP):")
	for operation, duration := range results {
		t.Logf("  %s: %v", operation, duration)
	}

	// Compare HTTP vs CLI for model listing
	httpTime := results["HTTP_GetModels"]
	cliTime := results["CLI_ListModels"]

	if httpTime < cliTime {
		t.Logf("HTTP is faster for listing models (%v vs %v)", httpTime, cliTime)
	} else {
		t.Logf("CLI is faster for listing models (%v vs %v)", cliTime, httpTime)
	}

	// Compare HTTP vs CLI for running models
	httpRunningTime := results["HTTP_GetRunningModels"]
	cliRunningTime := results["CLI_ListRunningModels"]

	if httpRunningTime < cliRunningTime {
		t.Logf("HTTP is faster for listing running models (%v vs %v)", httpRunningTime, cliRunningTime)
	} else {
		t.Logf("CLI is faster for listing running models (%v vs %v)", cliRunningTime, httpRunningTime)
	}
}

func TestCLICleanupVsHTTPCleanup(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping cleanup comparison test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
		t.Skip("No models available, skipping cleanup comparison test")
	}

	// Pick a small model for testing
	testModel := models[0].ID
	for _, model := range models {
		if model.ID == "phi3:3.8b" || model.ID == "llama3.2:3b" {
			testModel = model.ID
			break
		}
	}

	t.Logf("Testing cleanup comparison with model: %s", testModel)

	// Test 1: HTTP-based cleanup
	t.Log("Testing HTTP-based cleanup...")

	// Start model
	if err := StartModel(ctx, testModel); err != nil {
		t.Fatalf("Failed to start model: %v", err)
	}

	// Verify it's loaded
	loaded, err := IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded: %v", err)
	}
	if !loaded {
		t.Fatal("Model failed to load")
	}

	// Time HTTP cleanup
	start := time.Now()
	cleanupProcesses()
	httpCleanupTime := time.Since(start)

	// Give time for cleanup
	time.Sleep(2 * time.Second)

	// Check if model is still loaded
	loaded, err = IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded after HTTP cleanup: %v", err)
	}

	httpCleanupWorked := !loaded

	// Test 2: CLI-based cleanup
	t.Log("Testing CLI-based cleanup...")

	// Start model again
	if err := StartModel(ctx, testModel); err != nil {
		t.Fatalf("Failed to start model for CLI test: %v", err)
	}

	// Verify it's loaded
	loaded, err = IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded: %v", err)
	}
	if !loaded {
		t.Fatal("Model failed to load for CLI test")
	}

	// Time CLI cleanup
	start = time.Now()
	if err := CLICleanupProcesses(ctx); err != nil {
		t.Fatalf("CLI cleanup failed: %v", err)
	}
	cliCleanupTime := time.Since(start)

	// Give time for cleanup
	time.Sleep(2 * time.Second)

	// Check if model is still loaded
	loaded, err = IsModelLoaded(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is loaded after CLI cleanup: %v", err)
	}

	cliCleanupWorked := !loaded

	// Compare results
	t.Log("Cleanup Comparison Results:")
	t.Logf("  HTTP cleanup: %v (worked: %v)", httpCleanupTime, httpCleanupWorked)
	t.Logf("  CLI cleanup: %v (worked: %v)", cliCleanupTime, cliCleanupWorked)

	if httpCleanupWorked && cliCleanupWorked {
		if httpCleanupTime < cliCleanupTime {
			t.Logf("HTTP cleanup is faster and both work")
		} else {
			t.Logf("CLI cleanup is faster and both work")
		}
	} else if httpCleanupWorked && !cliCleanupWorked {
		t.Logf("HTTP cleanup works better (CLI cleanup failed)")
	} else if !httpCleanupWorked && cliCleanupWorked {
		t.Logf("CLI cleanup works better (HTTP cleanup failed)")
	} else {
		t.Logf("Both cleanup methods failed")
	}
}
