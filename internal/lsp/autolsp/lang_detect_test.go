package autolsp_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/caarlos0/testfs"
	"github.com/charmbracelet/crush/internal/lsp/autolsp"
)

func TestLangDetector(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		file     string
	}{
		{"Dart", []string{"dart", "yaml"}, "pubspec.yaml"},
		{"Elixir", []string{"elixir"}, "mix.exs"},
		{"Go", []string{"go"}, "go.mod"},
		{"Java-Gradle", []string{"java"}, "build.gradle"},
		{"Java-POM", []string{"java"}, "pom.xml"},
		{"JavaScript", []string{"javascript", "json"}, "package.json"},
		{"PHP", []string{"php", "json"}, "composer.json"},
		{"Python-Pyproject", []string{"python"}, "pyproject.toml"},
		{"Python-Requirements", []string{"python"}, "requirements.txt"},
		{"Python-SetupPy", []string{"python"}, "setup.py"},
		{"Ruby", []string{"ruby"}, "Gemfile"},
		{"Rust", []string{"rust"}, "Cargo.toml"},
		{"YAML-yaml", []string{"yaml"}, "config.yaml"},
		{"YAML-yml", []string{"yaml"}, "config.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfs := testfs.New(t)
			_ = tmpfs.MkdirAll(filepath.Dir(tt.file), 0o755)
			_ = tmpfs.WriteFile(tt.file, []byte(""), 0o644)

			d := autolsp.NewLangDetector(
				autolsp.LangDetectorWithFS(tmpfs),
			)
			langs := d.Detect()

			if !reflect.DeepEqual(tt.expected, collectNames(langs)) {
				t.Errorf("expected languages %s, got %s", tt.expected, collectNames(langs))
			}
		})
	}
}

func TestLangDetectorMultiple(t *testing.T) {
	tmpfs := testfs.New(t)

	{
		_ = tmpfs.MkdirAll("backend", 0o755)
		_ = tmpfs.WriteFile("backend/Gemfile", []byte(""), 0o644)
	}
	{
		_ = tmpfs.MkdirAll("frontend", 0o755)
		_ = tmpfs.WriteFile("frontend/package.json", []byte(""), 0o644)
	}

	d := autolsp.NewLangDetector(
		autolsp.LangDetectorWithFS(tmpfs),
	)
	expected := []string{"javascript", "ruby", "json"}
	langs := d.Detect()

	if !reflect.DeepEqual(expected, collectNames(langs)) {
		t.Errorf("expected languages %s, got %s", expected, collectNames(langs))
	}
}

func TestLangDetectorIgnoredDir(t *testing.T) {
	tmpfs := testfs.New(t)

	{
		_ = tmpfs.MkdirAll("node_modules", 0o755)
		_ = tmpfs.WriteFile("node_modules/package.json", []byte(""), 0o644)
	}

	d := autolsp.NewLangDetector(
		autolsp.LangDetectorWithFS(tmpfs),
	)
	expected := []string{}
	langs := d.Detect()

	if !reflect.DeepEqual(expected, collectNames(langs)) {
		t.Errorf("expected languages %s, got %s", expected, collectNames(langs))
	}
}

func TestLangDetectorThisProject(t *testing.T) {
	d := autolsp.NewLangDetector(
		autolsp.LangDetectorWithFS(os.DirFS("../../..")),
	)
	expected := []string{
		"go",
		"yaml",
		"json",
		"bash",
	}
	langs := d.Detect()

	if !reflect.DeepEqual(expected, collectNames(langs)) {
		t.Errorf("expected languages %s, got %s", expected, collectNames(langs))
	}
}

func collectNames(langs []autolsp.Lang) []string {
	names := make([]string, len(langs))
	for i, lang := range langs {
		names[i] = lang.Name
	}
	return names
}
