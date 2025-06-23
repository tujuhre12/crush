package autolsp

type Lang struct {
	Name              string
	WorkspacePatterns []string
	FilePatterns      []string
}

var Langs = []Lang{
	{
		Name:              "bash",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.sh", "*.bash"},
	},
	{
		Name:              "c",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.c"},
	},
	{
		Name:              "css",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.css"},
	},
	{
		Name:              "csharp",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.cs"},
	},
	{
		Name:              "dart",
		WorkspacePatterns: []string{"pubspec.yaml"},
		FilePatterns:      []string{"*.dart"},
	},
	{
		Name:              "docker",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"Dockerfile"},
	},
	{
		Name:              "elixir",
		WorkspacePatterns: []string{"mix.exs"},
		FilePatterns:      []string{"*.ex", "*.exs"},
	},
	{
		Name:              "go",
		WorkspacePatterns: []string{"go.mod"},
		FilePatterns:      []string{"*.go"},
	},
	{
		Name:              "html",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.html", "*.htm"},
	},
	{
		Name:              "json",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.json"},
	},
	{
		Name:              "java",
		WorkspacePatterns: []string{"build.gradle", "pom.xml"},
		FilePatterns:      []string{"*.java"},
	},
	{
		Name:              "javascript",
		WorkspacePatterns: []string{"package.json"},
		FilePatterns:      []string{"*.js", "*.jsx", "*.mjs", "*.cjs"},
	},
	{
		Name:              "lua",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.lua"},
	},
	{
		Name:              "php",
		WorkspacePatterns: []string{"composer.json"},
		FilePatterns:      []string{"*.php"},
	},
	{
		Name:              "python",
		WorkspacePatterns: []string{"pyproject.toml", "requirements.txt", "setup.py"},
		FilePatterns:      []string{"*.py"},
	},
	{
		Name:              "ruby",
		WorkspacePatterns: []string{"Gemfile", "*.gemspec"},
		FilePatterns:      []string{"*.rb", "*.rake", "*.gemspec"},
	},
	{
		Name:              "rust",
		WorkspacePatterns: []string{"Cargo.toml"},
		FilePatterns:      []string{"*.rs"},
	},
	{
		Name:              "svelte",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.svelte"},
	},
	{
		Name:              "typescript",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.ts", "*.tsx"},
	},
	{
		Name:              "vue",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.vue"},
	},
	{
		Name:              "yaml",
		WorkspacePatterns: nil,
		FilePatterns:      []string{"*.yaml", "*.yml"},
	},
}
