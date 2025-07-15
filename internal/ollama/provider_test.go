package ollama

import (
	"context"
	"testing"
	"time"
)

func TestGetProvider(t *testing.T) {
	if !IsInstalled() {
		t.Skip("Ollama is not installed, skipping GetProvider test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	t.Logf("✓ Provider: %s (ID: %s) with %d models",
		provider.Name, provider.ID, len(provider.Models))

	// Test model details
	for _, model := range provider.Models {
		t.Logf("  - %s (context: %d, max_tokens: %d, images: %v)",
			model.ID, model.ContextWindow, model.DefaultMaxTokens, model.SupportsImages)
	}

	// Cleanup
	defer func() {
		if processManager.crushStartedOllama {
			cleanup()
		}
	}()
}

func TestExtractModelFamily(t *testing.T) {
	tests := []struct {
		modelName string
		expected  string
	}{
		{"llama3.2:3b", "llama"},
		{"mistral:7b", "mistral"},
		{"gemma:2b", "gemma"},
		{"qwen2.5:14b", "qwen"},
		{"phi3:3.8b", "phi"},
		{"codellama:13b", "codellama"},
		{"llava:13b", "llava"},
		{"llama-vision:7b", "llama-vision"},
		{"unknown-model:1b", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := extractModelFamily(tt.modelName)
			if result != tt.expected {
				t.Errorf("extractModelFamily(%s) = %s, expected %s",
					tt.modelName, result, tt.expected)
			}
		})
	}
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
