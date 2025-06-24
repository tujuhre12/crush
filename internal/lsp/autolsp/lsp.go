package autolsp

type ServerName string

const (
	ServerBashLanguageServer       ServerName = "bash-language-server"
	ServerClangd                   ServerName = "clangd"
	ServerDart                     ServerName = "dart"
	ServerDeno                     ServerName = "deno"
	ServerDockerLangserver         ServerName = "docker-langserver"
	ServerElixirLS                 ServerName = "elixir-ls"
	ServerGopls                    ServerName = "gopls"
	ServerIntelephense             ServerName = "intelephense"
	ServerJDTLS                    ServerName = "jdtls"
	ServerJediLanguageServer       ServerName = "jedi-language-server"
	ServerLuaLanguageServer        ServerName = "lua-language-server"
	ServerOmnisharp                ServerName = "omnisharp"
	ServerPylsp                    ServerName = "pylsp"
	ServerPyright                  ServerName = "pyright"
	ServerRustAnalyzer             ServerName = "rust-analyzer"
	ServerSolargraph               ServerName = "solargraph"
	ServerSvelteserver             ServerName = "svelteserver"
	ServerTypeScriptLanguageServer ServerName = "typescript-language-server"
	ServerVSCodeCSSLanguageServer  ServerName = "vscode-css-language-server"
	ServerVSCodeHTMLLanguageServer ServerName = "vscode-html-language-server"
	ServerVSCodeJSONLanguageServer ServerName = "vscode-json-language-server"
	ServerVLS                      ServerName = "vls"
	ServerYAMLLanguageServer       ServerName = "yaml-language-server"
)

type Server struct {
	Name           ServerName
	Args           []string
	InstallCmd     string
	InstallWebsite string
	Langs          []LangName
}

var Servers = []Server{
	{
		Name:           ServerBashLanguageServer,
		Args:           []string{"start"},
		InstallCmd:     "npm install -g bash-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangBash},
	},
	{
		Name:           ServerClangd,
		InstallCmd:     "",
		InstallWebsite: "https://clangd.llvm.org/installation.html",
		Langs:          []LangName{LangC},
	},
	{
		Name:           ServerDart,
		Args:           []string{"language-server"},
		InstallCmd:     "",
		InstallWebsite: "https://dart.dev/get-dart",
		Langs:          []LangName{LangDart},
	},
	{
		Name:           ServerDeno,
		Args:           []string{"lsp"},
		InstallCmd:     "",
		InstallWebsite: "https://deno.com/#installation",
		Langs:          []LangName{LangJavaScript, LangTypeScript},
	},
	{
		Name:           ServerDockerLangserver,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g dockerfile-language-server-nodejs",
		InstallWebsite: "",
		Langs:          []LangName{LangDocker},
	},
	{
		Name:           ServerElixirLS,
		InstallCmd:     "",
		InstallWebsite: "https://github.com/elixir-lsp/elixir-ls#installation",
		Langs:          []LangName{LangElixir},
	},
	{
		Name:           ServerGopls,
		InstallCmd:     "go install golang.org/x/tools/gopls@latest",
		InstallWebsite: "",
		Langs:          []LangName{LangGo},
	},
	{
		Name:           ServerIntelephense,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g intelephense",
		InstallWebsite: "",
		Langs:          []LangName{LangPHP},
	},
	{
		Name:           ServerJDTLS,
		InstallCmd:     "",
		InstallWebsite: "https://github.com/eclipse/eclipse.jdt.ls",
		Langs:          []LangName{LangJava},
	},
	{
		Name:           ServerJediLanguageServer,
		InstallCmd:     "pip install jedi-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangPython},
	},
	{
		Name:           ServerLuaLanguageServer,
		InstallCmd:     "",
		InstallWebsite: "https://github.com/LuaLS/lua-language-server/wiki/Getting-Started",
		Langs:          []LangName{LangLua},
	},
	{
		Name:           ServerOmnisharp,
		Args:           []string{"--languageserver"},
		InstallCmd:     "npm install -g omnisharp-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangCSharp},
	},
	{
		Name:           ServerPylsp,
		Args:           []string{},
		InstallCmd:     "pip install python-lsp-server",
		InstallWebsite: "",
		Langs:          []LangName{LangPython},
	},
	{
		Name:           ServerPyright,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g pyright",
		InstallWebsite: "",
		Langs:          []LangName{LangPython},
	},
	{
		Name:           ServerRustAnalyzer,
		Args:           []string{},
		InstallCmd:     "rustup component add rust-analyzer",
		InstallWebsite: "",
		Langs:          []LangName{LangRust},
	},
	{
		Name:           ServerSolargraph,
		Args:           []string{"stdio"},
		InstallCmd:     "gem install solargraph",
		InstallWebsite: "",
		Langs:          []LangName{LangRuby},
	},
	{
		Name:           ServerSvelteserver,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g svelte-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangSvelte},
	},
	{
		Name:           ServerTypeScriptLanguageServer,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g typescript-language-server typescript",
		InstallWebsite: "",
		Langs:          []LangName{LangTypeScript},
	},
	{
		Name:           ServerVSCodeCSSLanguageServer,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []LangName{LangCSS},
	},
	{
		Name:           ServerVSCodeHTMLLanguageServer,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []LangName{LangHTML},
	},
	{
		Name:           ServerVSCodeJSONLanguageServer,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g vscode-langservers-extracted",
		InstallWebsite: "",
		Langs:          []LangName{LangJSON},
	},
	{
		Name:           ServerVLS,
		Args:           []string{},
		InstallCmd:     "npm install -g @volar/vue-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangVue},
	},
	{
		Name:           ServerYAMLLanguageServer,
		Args:           []string{"--stdio"},
		InstallCmd:     "npm install -g yaml-language-server",
		InstallWebsite: "",
		Langs:          []LangName{LangYAML},
	},
}
