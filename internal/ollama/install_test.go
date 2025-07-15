package ollama

import (
	"testing"
)

func TestIsInstalled(t *testing.T) {
	installed := IsInstalled()

	if installed {
		t.Log("✓ Ollama is installed on this system")
	} else {
		t.Log("✗ Ollama is not installed on this system")
	}

	// This is informational - doesn't fail
}

func BenchmarkIsInstalled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsInstalled()
	}
}
