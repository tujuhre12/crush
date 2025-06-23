package autolsp

type LSP string

const (
	ServerBashLanguageServer       LSP = "bash-language-server"
	ServerClangd                       = "clangd"
	ServerDart                         = "dart"
	ServerDeno                         = "deno"
	ServerDockerLangServer             = "docker-langserver"
	ServerElixirLS                     = "elixir-ls"
	ServerGopls                        = "gopls"
	ServerIntelephense                 = "intelephense"
	ServerJdtls                        = "jdtls"
	ServerJediLanguageServer           = "jedi-language-server"
	ServerLuaLanguageServer            = "lua-language-server"
	ServerOmnisharp                    = "omnisharp"
	ServerPylsp                        = "pylsp"
	ServerPyright                      = "pyright"
	ServerRustAnalyzer                 = "rust-analyzer"
	ServerSolargraph                   = "solargraph"
	ServerSvelteServer                 = "svelteserver"
	ServerTypescriptLanguageServer     = "typescript-language-server"
	ServerVSCodeCSSLanguageServer      = "vscode-css-language-server"
	ServerVSCodeHTMLLanguageServer     = "vscode-html-language-server"
	ServerVSCodeJSONLanguageServer     = "vscode-json-language-server"
	ServerVLS                          = "vls"
	ServerYAMLLanguageServer           = "yaml-language-server"
)

type Server struct {
	LSP            LSP
	Executable     string
	Args           []string
	InstallCmd     string
	InstallWebsite string
	Langs          []Lang
}

var Servers = []Server{
	{
		LSP:        ServerBashLanguageServer,
		Executable: "bash-language-server",
		Args:       []string{"start"},
		InstallCmd: "npm install -g bash-language-server",
		Langs:      []Lang{Bash},
	},
	{
		LSP:            ServerClangd,
		Executable:     "clangd",
		InstallWebsite: "https://clangd.llvm.org/installation.html",
		Langs:          []Lang{C},
	},
	{
		LSP:            ServerDart,
		Executable:     "dart",
		Args:           []string{"language-server"},
		InstallWebsite: "https://dart.dev/get-dart",
		Langs:          []Lang{Dart},
	},
	{
		LSP:            ServerDeno,
		Executable:     "deno",
		Args:           []string{"lsp"},
		InstallWebsite: "https://deno.com/#installation",
		Langs:          []Lang{JavaScript, TypeScript},
	},
	{
		LSP:        ServerDockerLangServer,
		Executable: "docker-langserver",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g dockerfile-language-server-nodejs",
		Langs:      []Lang{Docker},
	},
	{
		LSP:            ServerElixirLS,
		Executable:     "elixir-ls",
		InstallWebsite: "https://github.com/elixir-lsp/elixir-ls#installation",
		Langs:          []Lang{Elixir},
	},
	{
		LSP:        ServerGopls,
		Executable: "gopls",
		InstallCmd: "go install golang.org/x/tools/gopls@latest",
		Langs:      []Lang{Go},
	},
	{
		LSP:        ServerIntelephense,
		Executable: "intelephense",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g intelephense",
		Langs:      []Lang{PHP},
	},
	{
		LSP:            ServerJdtls,
		Executable:     "jdtls",
		InstallWebsite: "https://github.com/eclipse/eclipse.jdt.ls",
		Langs:          []Lang{Java},
	},
	{
		LSP:        ServerJediLanguageServer,
		Executable: "jedi-language-server",
		InstallCmd: "pip install jedi-language-server",
		Langs:      []Lang{Python},
	},
	{
		LSP:            ServerLuaLanguageServer,
		Executable:     "lua-language-server",
		InstallWebsite: "https://github.com/LuaLS/lua-language-server/wiki/Getting-Started",
		Langs:          []Lang{Lua},
	},
	{
		LSP:        ServerOmnisharp,
		Executable: "omnisharp",
		Args:       []string{"--languageserver"},
		InstallCmd: "npm install -g omnisharp-language-server",
		Langs:      []Lang{CSharp},
	},
	{
		LSP:        ServerPylsp,
		Executable: "pylsp",
		Args:       []string{},
		InstallCmd: "pip install python-lsp-server",
		Langs:      []Lang{Python},
	},
	{
		LSP:        ServerPyright,
		Executable: "pyright",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g pyright",
		Langs:      []Lang{Python},
	},
	{
		LSP:        ServerRustAnalyzer,
		Executable: "rust-analyzer",
		Args:       []string{},
		InstallCmd: "rustup component add rust-analyzer",
		Langs:      []Lang{Rust},
	},
	{
		LSP:        ServerSolargraph,
		Executable: "solargraph",
		Args:       []string{"stdio"},
		InstallCmd: "gem install solargraph",
		Langs:      []Lang{Ruby},
	},
	{
		LSP:        ServerSvelteServer,
		Executable: "svelteserver",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g svelte-language-server",
		Langs:      []Lang{JavaScript, TypeScript},
	},
	{
		LSP:        ServerTypescriptLanguageServer,
		Executable: "typescript-language-server",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g typescript-language-server typescript",
		Langs:      []Lang{JavaScript, TypeScript},
	},
	{
		LSP:        ServerVSCodeCSSLanguageServer,
		Executable: "vscode-css-language-server",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g vscode-langservers-extracted",
		Langs:      []Lang{CSS},
	},
	{
		LSP:        ServerVSCodeHTMLLanguageServer,
		Executable: "vscode-html-language-server",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g vscode-langservers-extracted",
		Langs:      []Lang{HTML},
	},
	{
		LSP:        ServerVSCodeJSONLanguageServer,
		Executable: "vscode-json-language-server",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g vscode-langservers-extracted",
		Langs:      []Lang{JSON},
	},
	{
		LSP:        ServerVLS,
		Executable: "vue-language-server",
		Args:       []string{},
		InstallCmd: "npm install -g @volar/vue-language-server",
		Langs:      []Lang{JavaScript, TypeScript},
	},
	{
		LSP:        ServerYAMLLanguageServer,
		Executable: "yaml-language-server",
		Args:       []string{"--stdio"},
		InstallCmd: "npm install -g yaml-language-server",
		Langs:      []Lang{YAML},
	},
}
