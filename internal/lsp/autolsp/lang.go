package autolsp

type Lang int

const (
	C Lang = iota + 1
	Dart
	Elixir
	Go
	Java
	JavaScript
	PHP
	Python
	Ruby
	Rust
	TypeScript
)

var langNames = map[Lang]string{
	C:          "C",
	Dart:       "Dart",
	Elixir:     "Elixir",
	Go:         "Go",
	Java:       "Java",
	JavaScript: "JavaScript",
	PHP:        "PHP",
	Python:     "Python",
	Ruby:       "Ruby",
	Rust:       "Rust",
	TypeScript: "TypeScript",
}

func (l Lang) String() string {
	if name, ok := langNames[l]; ok {
		return name
	}
	return "Unknown"
}
