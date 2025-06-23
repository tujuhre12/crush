package autolsp

type Server struct {
	Name           string
	Args           []string
	InstallCmd     string
	InstallWebsite string
	Langs          []string
}

var Servers = []Server{
	{
		Name:           "bash-language-server",
		Args:           []string{"start"},
		InstallCmd:     "npm install -g bash-language-server",
		InstallWebsite: "",
		Langs:          []string{"bash"},
	},
	{
		Name:           "clangd",
		InstallCmd:     "",
		InstallWebsite: "https://clangd.llvm.org/installation.html",
		Langs:          []string{"c"},
	},
	{
		Name:           "dart",
		Args:           []string{"language-server"},
		InstallCmd:     "",
		InstallWebsite: "https://dart.dev/get-dart",
		Langs:          []string{"dart"},
	},
	{
		Name:           "deno",
		Args:           []string{"lsp"},
		InstallCmd:     "",
		InstallWebsite: "https://deno.com/#installation",
		Langs:          []string{"javascript", "typescript"},
	},
	{
		Name:           "docker-langserver",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g dockerfile-language-server-nodejs",
		InstallWebsite: "",
		Langs:          []string{"docker"},
	},
	{
		Name:           "elixir-ls",
		InstallCmd:     "",
		InstallWebsite: "https://github.com/elixir-lsp/elixir-ls#installation",
		Langs:          []string{"elixir"},
	},
	{
		Name:           "gopls",
		InstallCmd:     "go install golang.org/x/tools/gopls@latest",
		InstallWebsite: "",
		Langs:          []string{"go"},
	},
	{
		Name:           "intelephense",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g intelephense",
		InstallWebsite: "",
		Langs:          []string{"php"},
	},
	{
		Name:           "jdtls",
		InstallCmd:     "",
		InstallWebsite: "https://github.com/eclipse/eclipse.jdt.ls",
		Langs:          []string{"java"},
	},
	{
		Name:           "jedi-language-server",
		InstallCmd:     "pip install jedi-language-server",
		InstallWebsite: "",
		Langs:          []string{"python"},
	},
	{
		Name:           "lua-language-server",
		InstallCmd:     "",
		InstallWebsite: "https://github.com/LuaLS/lua-language-server/wiki/Getting-Started",
		Langs:          []string{"lua"},
	},
	{
		Name:           "omnisharp",
		Args:           []string{"--languageserver"},
		InstallCmd:     "npm install -g omnisharp-language-server",
		InstallWebsite: "",
		Langs:          []string{"csharp"},
	},
	{
		Name:           "pylsp",
		Args:           []string{},
		InstallCmd:     "pip install python-lsp-server",
		InstallWebsite: "",
		Langs:          []string{"python"},
	},
	{
		Name:           "pyright",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g pyright",
		InstallWebsite: "",
		Langs:          []string{"python"},
	},
	{
		Name:           "rust-analyzer",
		Args:           []string{},
		InstallCmd:     "rustup component add rust-analyzer",
		InstallWebsite: "",
		Langs:          []string{"rust"},
	},
	{
		Name:           "solargraph",
		Args:           []string{"stdio"},
		InstallCmd:     "gem install solargraph",
		InstallWebsite: "",
		Langs:          []string{"ruby"},
	},
	{
		Name:           "svelteserver",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g svelte-language-server",
		InstallWebsite: "",
		Langs:          []string{"svelte"},
	},
	{
		Name:           "typescript-language-server",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g typescript-language-server typescript",
		InstallWebsite: "",
		Langs:          []string{"typescript"},
	},
	{
		Name:           "vscode-css-language-server",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []string{"css"},
	},
	{
		Name:           "vscode-html-language-server",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []string{"html"},
	},
	{
		Name:           "vscode-json-language-server",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []string{"json"},
	},
	{
		Name:           "vls",
		Args:           []string{},
		InstallCmd:     "npm install -g @volar/vue-language-server",
		InstallWebsite: "",
		Langs:          []string{"vue"},
	},
	{
		Name:           "yaml-language-server",
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g yaml-language-server",
		InstallWebsite: "",
		Langs:          []string{"yaml"},
	},
}
