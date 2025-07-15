package ollama

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLI-based approach for Ollama operations
// These functions use the ollama CLI instead of HTTP requests

// CLIListModels lists available models using ollama CLI
func CLIListModels(ctx context.Context) ([]OllamaModel, error) {
	cmd := exec.CommandContext(ctx, "ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list models via CLI: %w", err)
	}

	return parseModelsList(string(output))
}

// parseModelsList parses the text output from 'ollama list'
func parseModelsList(output string) ([]OllamaModel, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected output format")
	}

	var models []OllamaModel
	// Skip the header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse each line: NAME ID SIZE MODIFIED
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := fields[0]
			models = append(models, OllamaModel{
				Name:  name,
				Model: name,
				Size:  0, // Size parsing from text is complex, skip for now
			})
		}
	}

	return models, nil
}

// CLIListRunningModels lists currently running models using ollama CLI
func CLIListRunningModels(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "ollama", "ps")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list running models via CLI: %w", err)
	}

	return parseRunningModelsList(string(output))
}

// parseRunningModelsList parses the text output from 'ollama ps'
func parseRunningModelsList(output string) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return []string{}, nil // No running models
	}

	var runningModels []string
	// Skip the header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse each line: NAME ID SIZE PROCESSOR UNTIL
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			name := fields[0]
			if name != "" {
				runningModels = append(runningModels, name)
			}
		}
	}

	return runningModels, nil
}

// CLIStopModel stops a specific model using ollama CLI
func CLIStopModel(ctx context.Context, modelName string) error {
	cmd := exec.CommandContext(ctx, "ollama", "stop", modelName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop model %s via CLI: %w", modelName, err)
	}
	return nil
}

// CLIStopAllModels stops all running models using ollama CLI
func CLIStopAllModels(ctx context.Context) error {
	// First get list of running models
	runningModels, err := CLIListRunningModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to get running models: %w", err)
	}

	// Stop each model individually
	for _, modelName := range runningModels {
		if err := CLIStopModel(ctx, modelName); err != nil {
			return fmt.Errorf("failed to stop model %s: %w", modelName, err)
		}
	}

	return nil
}

// CLIIsModelRunning checks if a specific model is running using ollama CLI
func CLIIsModelRunning(ctx context.Context, modelName string) (bool, error) {
	runningModels, err := CLIListRunningModels(ctx)
	if err != nil {
		return false, err
	}

	for _, running := range runningModels {
		if running == modelName {
			return true, nil
		}
	}

	return false, nil
}

// CLIStartModel starts a model using ollama CLI (similar to StartModel but using CLI)
func CLIStartModel(ctx context.Context, modelName string) error {
	// Use ollama run with a simple prompt that immediately exits
	cmd := exec.CommandContext(ctx, "ollama", "run", modelName, "--verbose", "hi")

	// Set a shorter timeout for the run command since we just want to load the model
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd = exec.CommandContext(runCtx, "ollama", "run", modelName, "hi")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start model %s via CLI: %w", modelName, err)
	}

	return nil
}

// CLIGetModelsCount returns the number of available models using CLI
func CLIGetModelsCount(ctx context.Context) (int, error) {
	models, err := CLIListModels(ctx)
	if err != nil {
		return 0, err
	}
	return len(models), nil
}

// Performance comparison helpers

// BenchmarkCLIvsHTTP compares CLI vs HTTP performance
func BenchmarkCLIvsHTTP(ctx context.Context) (map[string]time.Duration, error) {
	results := make(map[string]time.Duration)

	// Test HTTP approach
	start := time.Now()
	_, err := GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("HTTP GetModels failed: %w", err)
	}
	results["HTTP_GetModels"] = time.Since(start)

	// Test CLI approach
	start = time.Now()
	_, err = CLIListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("CLI ListModels failed: %w", err)
	}
	results["CLI_ListModels"] = time.Since(start)

	// Test HTTP running models
	start = time.Now()
	_, err = GetRunningModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("HTTP GetRunningModels failed: %w", err)
	}
	results["HTTP_GetRunningModels"] = time.Since(start)

	// Test CLI running models
	start = time.Now()
	_, err = CLIListRunningModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("CLI ListRunningModels failed: %w", err)
	}
	results["CLI_ListRunningModels"] = time.Since(start)

	return results, nil
}

// CLICleanupProcesses provides CLI-based cleanup (alternative to HTTP-based cleanup)
func CLICleanupProcesses(ctx context.Context) error {
	return CLIStopAllModels(ctx)
}
