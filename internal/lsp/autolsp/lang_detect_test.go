package autolsp_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/charmbracelet/crush/internal/lsp/autolsp"
	"github.com/spf13/afero"
)

func TestDetector(t *testing.T) {
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
			fs := afero.NewMemMapFs()
			{
				f, _ := fs.Create(tt.file)
				_ = f.Close()
			}

			d := autolsp.NewLangDetector(
				autolsp.LangDetectorWithFS(afero.NewIOFS(fs)),
			)
			langs := d.Detect()

			if !reflect.DeepEqual(tt.expected, collectNames(langs)) {
				t.Errorf("expected languages %s, got %s", tt.expected, collectNames(langs))
			}
		})
	}
}

func TestDetectorMultiple(t *testing.T) {
	fs := afero.NewMemMapFs()

	//nolint:gofumpt
	{
		_ = fs.Mkdir("backend", 0755)
		f, _ := fs.Create("backend/Gemfile")
		_ = f.Close()
	}
	//nolint:gofumpt
	{
		_ = fs.Mkdir("frontend", 0755)
		f, _ := fs.Create("frontend/package.json")
		_ = f.Close()
	}

	d := autolsp.NewLangDetector(
		autolsp.LangDetectorWithFS(afero.NewIOFS(fs)),
	)
	expected := []string{"javascript", "ruby", "json"}
	langs := d.Detect()

	if !reflect.DeepEqual(expected, collectNames(langs)) {
		t.Errorf("expected languages %s, got %s", expected, collectNames(langs))
	}
}

func TestDetectorIgnoredDir(t *testing.T) {
	fs := afero.NewMemMapFs()

	//nolint:gofumpt
	{
		_ = fs.Mkdir("node_modules", 0755)
		f, _ := fs.Create("node_modules/package.json")
		_ = f.Close()
	}

	d := autolsp.NewLangDetector(
		autolsp.LangDetectorWithFS(afero.NewIOFS(fs)),
	)
	expected := []string{}
	langs := d.Detect()

	if !reflect.DeepEqual(expected, collectNames(langs)) {
		t.Errorf("expected languages %s, got %s", expected, collectNames(langs))
	}
}

func TestDetectorThisProject(t *testing.T) {
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
