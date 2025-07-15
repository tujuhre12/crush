package ollama

import (
	"os/exec"
	"sync"
)

// OllamaModel represents a model returned by Ollama's API
type OllamaModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
	Digest     string `json:"digest"`
	Details    struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// OllamaTagsResponse represents the response from Ollama's /api/tags endpoint
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaRunningModel represents a model that is currently loaded in memory
type OllamaRunningModel struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Size    int64  `json:"size"`
	Digest  string `json:"digest"`
	Details struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
	ExpiresAt string `json:"expires_at"`
	SizeVRAM  int64  `json:"size_vram"`
}

// OllamaRunningModelsResponse represents the response from Ollama's /api/ps endpoint
type OllamaRunningModelsResponse struct {
	Models []OllamaRunningModel `json:"models"`
}

// ProcessManager manages Ollama processes started by Crush
type ProcessManager struct {
	mu                 sync.RWMutex
	processes          map[string]*exec.Cmd
	ollamaServer       *exec.Cmd // The main Ollama server process
	setupOnce          sync.Once
	crushStartedOllama bool // Track if Crush started the Ollama service
}
