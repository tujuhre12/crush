package heartbit

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/charmbracelet/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/rivo/uniseg"
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

func (h *Heartbit) Width() int {
	return lipgloss.Width(h.face)
}

func (h *Heartbit) Height() int {
	return lipgloss.Height(h.face)
}

func (h *Heartbit) Draw(scr uv.Screen, area uv.Rectangle) {
	grad := charmtone.Blend(h.Width(), charmtone.Tuna, charmtone.Blush)
	for y, line := range strings.Split(h.face, "\n") {
		seg := uniseg.NewGraphemes(line)
		var x int
		for seg.Next() {
			if seg.Str() == " " {
				x++
				continue
			}
			var style uv.Style
			style.Fg = grad[x]
			cell := uv.Cell{
				Style:   style,
				Content: seg.Str(),
				Width:   seg.Width(),
			}
			scr.SetCell(area.Min.X+x, area.Min.Y+y, &cell)
			x += cell.Width
		}
	}
}
