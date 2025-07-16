package heartbit

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
	tea "github.com/charmbracelet/bubbletea/v2"
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
	grad := charmtone.Blend(h.Width(), charmtone.Cheeky, charmtone.Dolly)
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

func (h *Heartbit) Render() string {
	scr := uv.NewScreenBuffer(h.Width(), h.Height())
	h.Draw(scr, scr.Bounds())
	return scr.Render()
}

func (h *Heartbit) Init() tea.Cmd {
	return nil
}

func (h *Heartbit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

func (h *Heartbit) View() string {
	return h.Render()
}

func (h *Heartbit) IsSectionHeader() bool {
	return true
}
