package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/crush/internal/fur/provider"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	requestTimeout   = 2 * time.Second
)

// IsRunning checks if Ollama is running by attempting to connect to its API
func IsRunning(ctx context.Context) bool {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", defaultOllamaURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetModels retrieves available models from Ollama
func GetModels(ctx context.Context) ([]provider.Model, error) {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", defaultOllamaURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var tagsResponse OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]provider.Model, len(tagsResponse.Models))
	for i, ollamaModel := range tagsResponse.Models {
		models[i] = provider.Model{
			ID:                 ollamaModel.Name,
			Model:              ollamaModel.Name,
			CostPer1MIn:        0, // Local models have no cost
			CostPer1MOut:       0,
			CostPer1MInCached:  0,
			CostPer1MOutCached: 0,
			ContextWindow:      getContextWindow(ollamaModel.Details.Family),
			DefaultMaxTokens:   4096,
			CanReason:          false,
			HasReasoningEffort: false,
			SupportsImages:     supportsImages(ollamaModel.Details.Family),
		}
	}

	return models, nil
}

// GetRunningModels returns models that are currently loaded in memory
func GetRunningModels(ctx context.Context) ([]OllamaRunningModel, error) {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", defaultOllamaURL+"/api/ps", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var psResponse OllamaRunningModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&psResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return psResponse.Models, nil
}

// IsModelLoaded checks if a specific model is currently loaded in memory
func IsModelLoaded(ctx context.Context, modelName string) (bool, error) {
	runningModels, err := GetRunningModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range runningModels {
		if model.Name == modelName {
			return true, nil
		}
	}

	return false, nil
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
