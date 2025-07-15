package ollama

import (
	"testing"
)

func TestIsInstalled(t *testing.T) {
	installed := IsInstalled()

	if installed {
		t.Log("Ollama is installed on this system")
	} else {
		t.Log("Ollama is not installed on this system")
	}

	// This test doesn't fail - it's informational
	// In a real scenario, you might want to skip other tests if Ollama is not installed
}

// Benchmark test for IsInstalled
func BenchmarkIsInstalled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsInstalled()
	}
}
