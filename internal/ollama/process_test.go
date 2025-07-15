package ollama

import (
	"context"
	"testing"
	"time"
)

func TestProcessManager(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping ProcessManager test")
	}

	// Test that processManager is initialized
	if processManager == nil {
		t.Fatal("processManager is nil")
	}

	if processManager.processes == nil {
		t.Fatal("processManager.processes is nil")
	}

	t.Log("✓ ProcessManager is properly initialized")
}

func TestCleanupProcesses(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping cleanup test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start Ollama service if not running
	wasRunning := IsRunning(ctx)
	if !wasRunning {
		t.Log("Starting Ollama service for cleanup test...")
		if err := StartOllamaService(ctx); err != nil {
			t.Fatalf("Failed to start Ollama service: %v", err)
		}

		// Verify it started
		if !IsRunning(ctx) {
			t.Fatal("Failed to start Ollama service")
		}

		// Test cleanup
		t.Log("Testing cleanup...")
		cleanupProcesses()

		// Give some time for cleanup
		time.Sleep(3 * time.Second)

		// Verify cleanup worked (service should be stopped)
		if IsRunning(ctx) {
			t.Error("Ollama service is still running after cleanup")
		} else {
			t.Log("✓ Cleanup successfully stopped Ollama service")
		}
	} else {
		t.Log("✓ Ollama was already running, skipping cleanup test to avoid disruption")
	}
}

func TestSetupProcessCleanup(t *testing.T) {
	// Test that setupProcessCleanup can be called without panicking
	// Note: We can't easily test signal handling in unit tests
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("setupProcessCleanup panicked: %v", r)
		}
	}()

	// This should not panic and should be safe to call multiple times
	setupProcessCleanup()
	setupProcessCleanup() // Should be safe due to sync.Once

	t.Log("✓ setupProcessCleanup completed without panic")
}

func TestProcessManagerThreadSafety(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping thread safety test")
	}

	// Test concurrent access to processManager
	done := make(chan bool)

	// Start multiple goroutines that access processManager
	for i := 0; i < 10; i++ {
		go func() {
			processManager.mu.RLock()
			_ = len(processManager.processes)
			processManager.mu.RUnlock()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Thread safety test timed out")
		}
	}

	t.Log("✓ ProcessManager thread safety test passed")
}
