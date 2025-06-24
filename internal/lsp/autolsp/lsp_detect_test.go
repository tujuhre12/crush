package autolsp_test

import (
	"os"
	"testing"

	"github.com/charmbracelet/crush/internal/lsp/autolsp"
)

func TestServerDetectorInstalled(t *testing.T) {
	lookPathFunc := func(name string) (string, error) {
		if name == "gopls" {
			return "/usr/local/bin/gopls", nil
		}
		return "", os.ErrNotExist
	}

	d := autolsp.NewServerDetector(
		autolsp.ServerDetectorWithLangs("go"),
		autolsp.ServerDetectorWithLookPathFunc(lookPathFunc),
	)
	installed, toBeInstalled := d.Detect()

	if len(installed) != 1 || installed[0].Name != "gopls" {
		t.Errorf("expected gopls to be installed, got %v", installed)
	}
	if len(toBeInstalled) != 0 {
		t.Errorf("expected no servers to be installed, got %v", toBeInstalled)
	}
}

func TestServerDetectorNotInstalled(t *testing.T) {
	lookPathFunc := func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	d := autolsp.NewServerDetector(
		autolsp.ServerDetectorWithLangs("go"),
		autolsp.ServerDetectorWithLookPathFunc(lookPathFunc),
	)
	installed, toBeInstalled := d.Detect()

	if len(installed) != 0 {
		t.Errorf("expected no servers to be installed, got %v", installed)
	}
	if len(toBeInstalled) != 1 || toBeInstalled[0].Name != "gopls" {
		t.Errorf("expected gopls to be in the list of servers to be installed, got %v", toBeInstalled)
	}
}
