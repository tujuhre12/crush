package heartbit

import (
	"strings"
	"unicode"

	"github.com/MakeNowJust/heredoc"
	uv "github.com/charmbracelet/ultraviolet"
)

var Primary = heredoc.Doc(`
    ▄▄▄▄▄▄▄▄    ▄▄▄▄▄▄▄▄
  ███████████  ███████████
████████████████████████████
████████████████████████████
██████████▀██████▀██████████
██████████ ██████ ██████████
▀▀██████▄████▄▄████▄██████▀▀
  ████████████████████████
    ████████████████████
       ▀▀██████████▀▀
           ▀▀▀▀▀▀
`)

type Heartbit struct {
	face string
}

func Standard() *Heartbit {
	return &Heartbit{
		face: Primary,
	}
}

func (h *Heartbit) Draw(scr uv.Screen, area uv.Rectangle) {
	for y, line := range strings.Split(h.face, "\n") {
		for x, r := range line {
			if unicode.IsSpace(r) {
				continue
			}
			var style uv.Style
			cell := uv.Cell{
				Style:   style,
				Content: string(r),
				Width:   0,
			}
			scr.SetCell(area.Min.X+x, area.Min.Y+y, &cell)
		}
	}
}
