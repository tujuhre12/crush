package ollama

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/crush/internal/fur/provider"
)

// IsRunning checks if Ollama is running by attempting to run a CLI command
func IsRunning(ctx context.Context) bool {
	_, err := CLIListModels(ctx)
	return err == nil
}

// GetModels retrieves available models from Ollama using CLI
func GetModels(ctx context.Context) ([]provider.Model, error) {
	ollamaModels, err := CLIListModels(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]provider.Model, len(ollamaModels))
	for i, ollamaModel := range ollamaModels {
		family := extractModelFamily(ollamaModel.Name)
		models[i] = provider.Model{
			ID:                 ollamaModel.Name,
			Model:              ollamaModel.Name,
			CostPer1MIn:        0, // Local models have no cost
			CostPer1MOut:       0,
			CostPer1MInCached:  0,
			CostPer1MOutCached: 0,
			ContextWindow:      getContextWindow(family),
			DefaultMaxTokens:   4096,
			CanReason:          false,
			HasReasoningEffort: false,
			SupportsImages:     supportsImages(family),
		}
	}

	return models, nil
}

// GetRunningModels returns models that are currently loaded in memory using CLI
func GetRunningModels(ctx context.Context) ([]OllamaRunningModel, error) {
	runningModelNames, err := CLIListRunningModels(ctx)
	if err != nil {
		return nil, err
	}

	var runningModels []OllamaRunningModel
	for _, name := range runningModelNames {
		runningModels = append(runningModels, OllamaRunningModel{
			Name: name,
		})
	}

	return runningModels, nil
}

// IsModelLoaded checks if a specific model is currently loaded in memory using CLI
func IsModelLoaded(ctx context.Context, modelName string) (bool, error) {
	return CLIIsModelRunning(ctx, modelName)
}

// GetProvider returns a provider.Provider for Ollama if it's running
func GetProvider(ctx context.Context) (*provider.Provider, error) {
	if !IsRunning(ctx) {
		return nil, fmt.Errorf("Ollama is not running")
	}

	models, err := GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	return &provider.Provider{
		Name:   "Ollama",
		ID:     "ollama",
		Models: models,
	}, nil
}

// extractModelFamily extracts the model family from a model name
func extractModelFamily(modelName string) string {
	// Extract the family from model names like "llama3.2:3b" -> "llama"
	parts := strings.Split(modelName, ":")
	if len(parts) > 0 {
		name := parts[0]
		// Handle cases like "llama3.2" -> "llama"
		if strings.HasPrefix(name, "llama") {
			return "llama"
		}
		if strings.HasPrefix(name, "mistral") {
			return "mistral"
		}
		if strings.HasPrefix(name, "gemma") {
			return "gemma"
		}
		if strings.HasPrefix(name, "qwen") {
			return "qwen"
		}
		if strings.HasPrefix(name, "phi") {
			return "phi"
		}
		if strings.HasPrefix(name, "codellama") {
			return "codellama"
		}
		if strings.Contains(name, "llava") {
			return "llava"
		}
		if strings.Contains(name, "vision") {
			return "llama-vision"
		}
	}
	return "unknown"
}

// getContextWindow returns an estimated context window based on model family
func getContextWindow(family string) int64 {
	switch family {
	case "llama":
		return 131072 // Llama 3.x context window
	case "mistral":
		return 32768
	case "gemma":
		return 8192
	case "qwen", "qwen2":
		return 131072
	case "phi":
		return 131072
	case "codellama":
		return 16384
	default:
		return 8192 // Conservative default
	}
}

// supportsImages returns whether a model family supports image inputs
func supportsImages(family string) bool {
	switch family {
	case "llama-vision", "llava":
		return true
	default:
		return false
	}
}
