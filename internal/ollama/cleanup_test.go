package ollama

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestProcessManagementWithRealModel(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping process management test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Start with a clean state
	originallyRunning := IsRunning(ctx)
	t.Logf("Ollama originally running: %v", originallyRunning)

	// If Ollama wasn't running, we'll start it and be responsible for cleanup
	var shouldCleanup bool
	if !originallyRunning {
		shouldCleanup = true
		t.Log("Starting Ollama service...")

		if err := StartService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}

		if !IsRunning(ctx) {
			t.Fatal("Started Ollama service but it's not running")
		}

		t.Log("✓ Ollama service started successfully")
	}

	// Get available models
	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping model loading test")
	}

	// Choose a test model (prefer smaller models)
	testModel := models[0].Name
	for _, model := range models {
		if model.Name == "phi3:3.8b" || model.Name == "llama3.2:3b" {
			testModel = model.Name
			break
		}
	}

	t.Logf("Testing with model: %s", testModel)

	// Test 1: Load model
	t.Log("Loading model...")
	startTime := time.Now()

	if err := EnsureModelLoaded(ctx, testModel); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	loadTime := time.Since(startTime)
	t.Logf("✓ Model loaded in %v", loadTime)

	// Verify model is running
	running, err := IsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running: %v", err)
	}

	if !running {
		t.Fatal("Model should be running but isn't")
	}

	t.Log("✓ Model is confirmed running")

	// Test 2: Immediate cleanup after loading
	t.Log("Testing immediate cleanup after model load...")

	cleanupStart := time.Now()
	cleanup()
	cleanupTime := time.Since(cleanupStart)

	t.Logf("✓ Cleanup completed in %v", cleanupTime)

	// Give cleanup time to take effect
	time.Sleep(2 * time.Second)

	// Test 3: Verify cleanup worked
	if shouldCleanup {
		// If we started Ollama, it should be stopped
		if IsRunning(ctx) {
			t.Error("❌ Ollama service should be stopped after cleanup but it's still running")
		} else {
			t.Log("✓ Ollama service properly stopped after cleanup")
		}
	} else {
		// If Ollama was already running, it should still be running but model should be stopped
		if !IsRunning(ctx) {
			t.Error("❌ Ollama service should still be running but it's stopped")
		} else {
			t.Log("✓ Ollama service still running (as expected)")

			// Check if model is still loaded
			running, err := IsModelRunning(ctx, testModel)
			if err != nil {
				t.Errorf("Failed to check model status after cleanup: %v", err)
			} else if running {
				t.Error("❌ Model should be stopped after cleanup but it's still running")
			} else {
				t.Log("✓ Model properly stopped after cleanup")
			}
		}
	}

	// Test 4: Test cleanup idempotency
	t.Log("Testing cleanup idempotency...")
	cleanup()
	cleanup()
	cleanup()
	t.Log("✓ Multiple cleanup calls handled safely")
}

func TestCleanupWithMockProcess(t *testing.T) {
	// Test cleanup mechanism with a mock process that simulates Ollama
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start mock process: %v", err)
	}

	pid := cmd.Process.Pid
	t.Logf("Started mock process with PID: %d", pid)

	// Simulate what happens in our process manager
	processManager.mu.Lock()
	processManager.ollamaProcess = cmd
	processManager.crushStartedOllama = true
	processManager.mu.Unlock()

	// Test cleanup
	t.Log("Testing cleanup with mock process...")
	stopOllamaService()

	// Verify process was terminated
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		t.Log("✓ Mock process was successfully terminated")
	} else {
		// Process might still be terminating
		time.Sleep(100 * time.Millisecond)
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Log("✓ Mock process was successfully terminated")
		} else {
			t.Error("❌ Mock process was not terminated")
		}
	}
}

func TestSetupCleanup(t *testing.T) {
	// Test that setupCleanup can be called without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("setupCleanup panicked: %v", r)
		}
	}()

	// This should not panic and should be safe to call multiple times
	setupCleanup()
	t.Log("✓ setupCleanup completed without panic")
}

func TestStopModel(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping stopModel test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure Ollama is running
	if err := EnsureRunning(ctx); err != nil {
		t.Fatalf("Failed to ensure Ollama is running: %v", err)
	}

	// Get available models
	models, err := GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Skip("No models available, skipping stopModel test")
	}

	testModel := models[0].Name
	t.Logf("Testing stop with model: %s", testModel)

	// Load the model first
	if err := EnsureModelLoaded(ctx, testModel); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	// Verify it's running
	running, err := IsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running: %v", err)
	}

	if !running {
		t.Fatal("Model should be running but isn't")
	}

	// Test stopping the model
	t.Log("Stopping model...")
	if err := stopModel(ctx, testModel); err != nil {
		t.Fatalf("Failed to stop model: %v", err)
	}

	// Give it time to stop
	time.Sleep(2 * time.Second)

	// Verify it's stopped
	running, err = IsModelRunning(ctx, testModel)
	if err != nil {
		t.Fatalf("Failed to check if model is running after stop: %v", err)
	}

	if running {
		t.Error("❌ Model should be stopped but it's still running")
	} else {
		t.Log("✓ Model successfully stopped")
	}

	// Cleanup
	defer func() {
		if processManager.crushStartedOllama {
			cleanup()
		}
	}()
}
