package autolsp

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

func (d *Detector) Detect() []Lang {
	priorities := make(map[string]int)

	_ = fs.WalkDir(d.fs, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if _, ok := dirsToIgnore[e.Name()]; ok {
			return filepath.SkipDir
		}
		for _, lang := range Langs {
			for _, pattern := range lang.WorkspacePatterns {
				if match, _ := filepath.Match(pattern, e.Name()); match {
					priorities[lang.Name] += 10
					break
				}
			}
			for _, pattern := range lang.FilePatterns {
				if match, _ := filepath.Match(pattern, e.Name()); match {
					priorities[lang.Name] += 1
					break
				}
			}
		}
		return nil
	})

	var langs []Lang
	for _, lang := range Langs {
		if priorities[lang.Name] > 0 {
			langs = append(langs, lang)
		}
	}
	slices.SortFunc(langs, func(a, b Lang) int {
		return priorities[b.Name] - priorities[a.Name]
	})
	return langs
}
