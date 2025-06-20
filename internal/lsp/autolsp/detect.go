package autolsp

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/x/exp/slice"
)

type Detector struct {
	fs fs.FS
}

type DetectorOption func(d *Detector)

func New(options ...DetectorOption) *Detector {
	d := Detector{}
	for _, opt := range options {
		opt(&d)
	}
	if d.fs == nil {
		d.fs = os.DirFS(".")
	}
	return &d
}

func WithFS(fs fs.FS) DetectorOption {
	return func(d *Detector) {
		d.fs = fs
	}
}

func WithDir(dir string) DetectorOption {
	return func(d *Detector) {
		d.fs = os.DirFS(dir)
	}
}

var detectPatterns = map[Lang][]string{
	C:          {},
	Dart:       {"pubspec.yaml"},
	Elixir:     {"mix.exs"},
	Go:         {"go.mod"},
	Java:       {"build.gradle", "pom.xml"},
	JavaScript: {"package.json"},
	PHP:        {"composer.json"},
	Python:     {"pyproject.toml", "requirements.txt", "setup.py"},
	Ruby:       {"Gemfile"},
	Rust:       {"Cargo.toml"},
	TypeScript: {},
}

func (d *Detector) Detect() (langs []Lang) {
	_ = fs.WalkDir(d.fs, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if _, ok := dirsToIgnore[e.Name()]; ok {
			return filepath.SkipDir
		}
		for lang, patterns := range detectPatterns {
			if slices.Contains(patterns, e.Name()) {
				langs = append(langs, lang)
				break
			}
		}
		return nil
	})
	return slice.Uniq(langs)
}
