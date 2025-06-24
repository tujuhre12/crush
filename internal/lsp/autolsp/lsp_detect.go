package autolsp

import (
	"os/exec"
	"slices"

	"github.com/charmbracelet/x/exp/slice"
)

// LookPathFunc is a function type that mimics exec.LookPath. Set this if you
// want to override the default behavior of looking for executables in the PATH.
type LookPathFunc func(string) (string, error)

type ServerDetector struct {
	langs        []string
	lookPathFunc LookPathFunc
}

type ServerDetectorOption func(d *ServerDetector)

func NewServerDetector(options ...ServerDetectorOption) *ServerDetector {
	d := ServerDetector{}
	for _, opt := range options {
		opt(&d)
	}
	if d.lookPathFunc == nil {
		d.lookPathFunc = exec.LookPath
	}
	return &d
}

func ServerDetectorWithLangs(langs ...string) ServerDetectorOption {
	return func(d *ServerDetector) {
		d.langs = langs
	}
}

func ServerDetectorWithLookPathFunc(lookPathFunc LookPathFunc) ServerDetectorOption {
	return func(d *ServerDetector) {
		d.lookPathFunc = lookPathFunc
	}
}

// Detect checks which language servers are installed and which ones need to be
// installed based on the provided languages. It's ordered by the priority,
// according to the order of languages in the `langs` slice.
func (d *ServerDetector) Detect() (installed, toBeInstalled []Server) {
	for _, server := range Servers {
		if !slice.ContainsAny(server.Langs, d.langs...) {
			continue
		}
		if _, err := d.lookPathFunc(server.Name); err == nil {
			installed = append(installed, server)
		} else {
			toBeInstalled = append(toBeInstalled, server)
		}
	}
	d.sortByPriority(installed)
	d.sortByPriority(toBeInstalled)
	return
}

func (d *ServerDetector) sortByPriority(servers []Server) {
	slices.SortStableFunc(servers, func(a, b Server) int {
		var (
			priorityA = slices.Index(d.langs, a.Langs[0])
			priorityB = slices.Index(d.langs, b.Langs[0])
		)
		switch {
		case priorityA == -1 && priorityB == -1:
			return 0
		case priorityA == -1:
			return 1
		case priorityB == -1:
			return -1
		default:
			return priorityA - priorityB
		}
	})
}
