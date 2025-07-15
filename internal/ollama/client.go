package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// httpClient creates a configured HTTP client
func httpClient() *http.Client {
	return &http.Client{
		Timeout: DefaultTimeout,
	}
}

// IsRunning checks if Ollama service is running
func IsRunning(ctx context.Context) bool {
	return isRunning(ctx, DefaultBaseURL)
}

// isRunning checks if Ollama is running at the specified URL
func isRunning(ctx context.Context, baseURL string) bool {
	client := httpClient()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
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

// GetModels retrieves all available models
func GetModels(ctx context.Context) ([]Model, error) {
	return getModels(ctx, DefaultBaseURL)
}

// getModels retrieves models from the specified URL
func getModels(ctx context.Context, baseURL string) ([]Model, error) {
	client := httpClient()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
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

	var response TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Models, nil
}

// GetRunningModels retrieves currently running models
func GetRunningModels(ctx context.Context) ([]RunningModel, error) {
	return getRunningModels(ctx, DefaultBaseURL)
}

// getRunningModels retrieves running models from the specified URL
func getRunningModels(ctx context.Context, baseURL string) ([]RunningModel, error) {
	client := httpClient()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/ps", nil)
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

	var response ProcessStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Models, nil
}

// IsModelRunning checks if a specific model is currently running
func IsModelRunning(ctx context.Context, modelName string) (bool, error) {
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

// LoadModel loads a model into memory by sending a simple request
func LoadModel(ctx context.Context, modelName string) error {
	return loadModel(ctx, DefaultBaseURL, modelName)
}

// loadModel loads a model at the specified URL
func loadModel(ctx context.Context, baseURL, modelName string) error {
	client := httpClient()

	reqBody := GenerateRequest{
		Model:  modelName,
		Prompt: "hi",
		Stream: false,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to load model, status: %d", resp.StatusCode)
	}

	return nil
}
