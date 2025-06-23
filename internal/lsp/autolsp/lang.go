package autolsp

type Lang int

const (
	Bash Lang = iota + 1
	C
	CSS
	CSharp
	Dart
	Docker
	Elixir
	Go
	HTML
	JSON
	Java
	JavaScript
	Lua
	PHP
	Python
	Ruby
	Rust
	TypeScript
	Vue
	YAML
)

var langNames = map[Lang]string{
	Bash:       "Bash",
	C:          "C",
	CSharp:     "C#",
	Dart:       "Dart",
	Docker:     "Docker",
	Elixir:     "Elixir",
	Go:         "Go",
	Java:       "Java",
	JavaScript: "JavaScript",
	Lua:        "Lua",
	PHP:        "PHP",
	Python:     "Python",
	Ruby:       "Ruby",
	Rust:       "Rust",
	TypeScript: "TypeScript",
	Vue:        "Vue",
	YAML:       "YAML",
}

func (l Lang) String() string {
	if name, ok := langNames[l]; ok {
		return name
	}
	return "Unknown"
}
