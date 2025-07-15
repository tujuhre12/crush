package ollama

import (
	"context"
	"fmt"
	"strings"
)

// ProviderModel represents a model in the provider format
type ProviderModel struct {
	ID                 string
	Model              string
	CostPer1MIn        float64
	CostPer1MOut       float64
	CostPer1MInCached  float64
	CostPer1MOutCached float64
	ContextWindow      int64
	DefaultMaxTokens   int64
	CanReason          bool
	HasReasoningEffort bool
	SupportsImages     bool
}

// Provider represents an Ollama provider
type Provider struct {
	Name   string
	ID     string
	Models []ProviderModel
}

// GetProvider returns a Provider for Ollama
func GetProvider(ctx context.Context) (*Provider, error) {
	if err := EnsureRunning(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure Ollama is running: %w", err)
	}

	models, err := GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	providerModels := make([]ProviderModel, len(models))
	for i, model := range models {
		family := extractModelFamily(model.Name)
		providerModels[i] = ProviderModel{
			ID:                 model.Name,
			Model:              model.Name,
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

	return &Provider{
		Name:   "Ollama",
		ID:     "ollama",
		Models: providerModels,
	}, nil
}

// extractModelFamily extracts the model family from a model name
func extractModelFamily(modelName string) string {
	// Extract the family from model names like "llama3.2:3b" -> "llama"
	parts := strings.Split(modelName, ":")
	if len(parts) > 0 {
		name := strings.ToLower(parts[0])

		// Handle various model families in specific order
		switch {
		case strings.Contains(name, "llama-vision"):
			return "llama-vision"
		case strings.Contains(name, "codellama"):
			return "codellama"
		case strings.Contains(name, "llava"):
			return "llava"
		case strings.Contains(name, "llama"):
			return "llama"
		case strings.Contains(name, "mistral"):
			return "mistral"
		case strings.Contains(name, "gemma"):
			return "gemma"
		case strings.Contains(name, "qwen"):
			return "qwen"
		case strings.Contains(name, "phi"):
			return "phi"
		case strings.Contains(name, "vision"):
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
	case "qwen":
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
