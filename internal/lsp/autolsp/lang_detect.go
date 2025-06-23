package autolsp

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

type LangDetector struct {
	fs fs.FS
}

type LangDetectorOption func(d *LangDetector)

func NewLangDetector(options ...LangDetectorOption) *LangDetector {
	d := LangDetector{}
	for _, opt := range options {
		opt(&d)
	}
	if d.fs == nil {
		d.fs = os.DirFS(".")
	}
	return &d
}

func LangDetectorWithFS(fs fs.FS) LangDetectorOption {
	return func(d *LangDetector) {
		d.fs = fs
	}
}

func LangDetectorWithDir(dir string) LangDetectorOption {
	return func(d *LangDetector) {
		d.fs = os.DirFS(dir)
	}
}

func (d *LangDetector) Detect() []Lang {
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
