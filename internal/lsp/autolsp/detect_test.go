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
		expected []autolsp.Lang
		file     string
	}{
		{"Dart", []autolsp.Lang{autolsp.Dart, autolsp.YAML}, "pubspec.yaml"},
		{"Elixir", []autolsp.Lang{autolsp.Elixir}, "mix.exs"},
		{"Go", []autolsp.Lang{autolsp.Go}, "go.mod"},
		{"Java-Gradle", []autolsp.Lang{autolsp.Java}, "build.gradle"},
		{"Java-POM", []autolsp.Lang{autolsp.Java}, "pom.xml"},
		{"JavaScript", []autolsp.Lang{autolsp.JavaScript}, "package.json"},
		{"PHP", []autolsp.Lang{autolsp.PHP}, "composer.json"},
		{"Python-Pyproject", []autolsp.Lang{autolsp.Python}, "pyproject.toml"},
		{"Python-Requirements", []autolsp.Lang{autolsp.Python}, "requirements.txt"},
		{"Python-SetupPy", []autolsp.Lang{autolsp.Python}, "setup.py"},
		{"Ruby", []autolsp.Lang{autolsp.Ruby}, "Gemfile"},
		{"Rust", []autolsp.Lang{autolsp.Rust}, "Cargo.toml"},
		{"YAML-yaml", []autolsp.Lang{autolsp.YAML}, "config.yaml"},
		{"YAML-yml", []autolsp.Lang{autolsp.YAML}, "config.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			{
				f, _ := fs.Create(tt.file)
				_ = f.Close()
			}

			d := autolsp.New(
				autolsp.WithFS(afero.NewIOFS(fs)),
			)
			langs := d.Detect()

			if !reflect.DeepEqual(langs, tt.expected) {
				t.Errorf("expected languages %s, got %s", tt.expected, langs)
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

	d := autolsp.New(
		autolsp.WithFS(afero.NewIOFS(fs)),
	)
	expected := []autolsp.Lang{
		autolsp.JavaScript,
		autolsp.Ruby,
	}
	langs := d.Detect()

	if !reflect.DeepEqual(langs, expected) {
		t.Errorf("expected languages %s, got %s", expected, langs)
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

	d := autolsp.New(
		autolsp.WithFS(afero.NewIOFS(fs)),
	)
	expected := []autolsp.Lang{}
	langs := d.Detect()

	if !reflect.DeepEqual(langs, expected) {
		t.Errorf("expected languages %s, got %s", expected, langs)
	}
}

func TestDetectorThisProject(t *testing.T) {
	d := autolsp.New(
		autolsp.WithFS(os.DirFS("../../..")),
	)
	expected := []autolsp.Lang{
		autolsp.Go,
		autolsp.YAML,
	}
	langs := d.Detect()

	if !reflect.DeepEqual(langs, expected) {
		t.Errorf("expected languages %s, got %s", expected, langs)
	}
}
