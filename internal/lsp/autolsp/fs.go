package autolsp

var dirsToIgnore = map[string]struct{}{
	".git":         {},
	".github":      {},
	".gitlab":      {},
	".hg":          {},
	".idea":        {},
	".svn":         {},
	".task":        {},
	".vscode":      {},
	"build":        {},
	"dist":         {},
	"node_modules": {},
	"vendor":       {},
}
