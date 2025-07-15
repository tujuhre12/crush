package ollama

import (
	"os/exec"
	"sync"
	"time"
)

// Constants for configuration
const (
	DefaultBaseURL      = "http://localhost:11434"
	DefaultTimeout      = 30 * time.Second
	ServiceStartTimeout = 15 * time.Second
	ModelLoadTimeout    = 60 * time.Second
)

// Model represents an Ollama model
type Model struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	ModifiedAt time.Time `json:"modified_at"`
	Details    struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// RunningModel represents a model currently loaded in memory
type RunningModel struct {
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	ExpiresAt time.Time `json:"expires_at"`
	SizeVRAM  int64     `json:"size_vram"`
	Details   struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// TagsResponse represents the response from /api/tags
type TagsResponse struct {
	Models []Model `json:"models"`
}

// ProcessStatusResponse represents the response from /api/ps
type ProcessStatusResponse struct {
	Models []RunningModel `json:"models"`
}

// GenerateRequest represents a request to /api/generate
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ProcessManager manages Ollama processes started by Crush
type ProcessManager struct {
	mu                 sync.RWMutex
	ollamaProcess      *exec.Cmd
	crushStartedOllama bool
	setupOnce          sync.Once
}
