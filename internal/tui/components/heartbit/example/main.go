package main

import (
	"github.com/charmbracelet/crush/internal/tui/components/heartbit"
	"github.com/charmbracelet/lipgloss/v2"
)

func main() {
	hb := heartbit.Primary
	lipgloss.Println(hb)
}
