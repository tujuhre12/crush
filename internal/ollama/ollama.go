package ollama

import (
	"os/exec"
)

// IsInstalled checks if Ollama is installed on the system
func IsInstalled() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}
