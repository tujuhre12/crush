package ollama

import (
	"os/exec"
	"sync"
)

// OllamaModel represents a model parsed from Ollama CLI output
type OllamaModel struct {
	Name  string
	Model string
	Size  int64
}

// OllamaRunningModel represents a model that is currently loaded in memory
type OllamaRunningModel struct {
	Name string
}

// ProcessManager manages Ollama processes started by Crush
type ProcessManager struct {
	mu                 sync.RWMutex
	processes          map[string]*exec.Cmd
	ollamaServer       *exec.Cmd // The main Ollama server process
	setupOnce          sync.Once
	crushStartedOllama bool // Track if Crush started the Ollama service
}
