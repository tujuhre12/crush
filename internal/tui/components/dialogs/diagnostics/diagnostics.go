package diagnostics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/lsp/protocol"
	"github.com/charmbracelet/crush/internal/tui/components/completions"
	"github.com/charmbracelet/crush/internal/tui/components/core"
	"github.com/charmbracelet/crush/internal/tui/components/core/list"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs/commands"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
	"github.com/charmbracelet/lipgloss/v2"
)

const (
	DiagnosticsDialogID dialogs.DialogID = "diagnostics"

	defaultWidth = 80
)

// DiagnosticItem represents a diagnostic entry
type DiagnosticItem struct {
	FilePath   string
	Diagnostic protocol.Diagnostic
	LSPName    string
}

// DiagnosticsDialog interface for the diagnostics dialog
type DiagnosticsDialog interface {
	dialogs.DialogModel
}

type diagnosticsDialogCmp struct {
	width   int
	wWidth  int
	wHeight int

	diagnosticsList list.ListModel
	keyMap          KeyMap
	help            help.Model
	lspClients      map[string]*lsp.Client
}

func NewDiagnosticsDialogCmp(lspClients map[string]*lsp.Client) DiagnosticsDialog {
	listKeyMap := list.DefaultKeyMap()
	keyMap := DefaultKeyMap()

	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.HalfPageDown.SetEnabled(false)
	listKeyMap.HalfPageUp.SetEnabled(false)
	listKeyMap.Home.SetEnabled(false)
	listKeyMap.End.SetEnabled(false)

	listKeyMap.DownOneItem = keyMap.Next
	listKeyMap.UpOneItem = keyMap.Previous

	t := styles.CurrentTheme()
	inputStyle := t.S().Base.Padding(0, 1, 0, 1)
	diagnosticsList := list.New(
		list.WithFilterable(true),
		list.WithKeyMap(listKeyMap),
		list.WithInputStyle(inputStyle),
		list.WithWrapNavigation(true),
	)
	help := help.New()
	help.Styles = t.S().Help

	return &diagnosticsDialogCmp{
		diagnosticsList: diagnosticsList,
		width:           defaultWidth,
		keyMap:          DefaultKeyMap(),
		help:            help,
		lspClients:      lspClients,
	}
}

func (d *diagnosticsDialogCmp) Init() tea.Cmd {
	d.loadDiagnostics()
	return d.diagnosticsList.Init()
}

func (d *diagnosticsDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.loadDiagnostics()
		return d, d.diagnosticsList.SetSize(d.listWidth(), d.listHeight())
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.Select):
			selectedItemInx := d.diagnosticsList.SelectedIndex()
			if selectedItemInx == list.NoSelection {
				return d, nil
			}
			items := d.diagnosticsList.Items()
			if selectedItem, ok := items[selectedItemInx].(completions.CompletionItem); ok {
				if diagItem, ok := selectedItem.Value().(DiagnosticItem); ok {
					// Open the file at the diagnostic location
					_ = fmt.Sprintf("%s:%d:%d", 
						diagItem.FilePath, 
						diagItem.Diagnostic.Range.Start.Line+1, 
						diagItem.Diagnostic.Range.Start.Character+1)
					
					return d, tea.Sequence(
						util.CmdHandler(dialogs.CloseDialogMsg{}),
						// You might want to add a message to open the file/location
						// For now, we'll just close the dialog
					)
				}
			}
			return d, nil
		case key.Matches(msg, d.keyMap.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})
		default:
			u, cmd := d.diagnosticsList.Update(msg)
			d.diagnosticsList = u.(list.ListModel)
			return d, cmd
		}
	}
	return d, nil
}

func (d *diagnosticsDialogCmp) View() tea.View {
	t := styles.CurrentTheme()
	listView := d.diagnosticsList.View()
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title("Diagnostics", d.width-5)),
		listView.String(),
		"",
		t.S().Base.Width(d.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(d.help.View(d.keyMap)),
	)
	v := tea.NewView(d.style().Render(content))
	if listView.Cursor() != nil {
		c := d.moveCursor(listView.Cursor())
		v.SetCursor(c)
	}
	return v
}

func (d *diagnosticsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(d.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (d *diagnosticsDialogCmp) listWidth() int {
	return defaultWidth - 2
}

func (d *diagnosticsDialogCmp) listHeight() int {
	listHeight := len(d.diagnosticsList.Items()) + 2 + 4 // height based on items + 2 for the input + 4 for the sections
	return min(listHeight, d.wHeight/2)
}

func (d *diagnosticsDialogCmp) Position() (int, int) {
	row := d.wHeight/4 - 2 // just a bit above the center
	col := d.wWidth / 2
	col -= d.width / 2
	return row, col
}

func (d *diagnosticsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + 3 // Border + title
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

func (d *diagnosticsDialogCmp) ID() dialogs.DialogID {
	return DiagnosticsDialogID
}

func (d *diagnosticsDialogCmp) loadDiagnostics() {
	diagnosticItems := []util.Model{}
	
	// Group diagnostics by LSP
	lspDiagnostics := make(map[string][]DiagnosticItem)
	
	for lspName, client := range d.lspClients {
		diagnostics := client.GetDiagnostics()
		var items []DiagnosticItem
		
		for location, diags := range diagnostics {
			for _, diag := range diags {
				items = append(items, DiagnosticItem{
					FilePath:   location.Path(),
					Diagnostic: diag,
					LSPName:    lspName,
				})
			}
		}
		
		// Sort diagnostics by severity (errors first) then by file path
		sort.Slice(items, func(i, j int) bool {
			iSeverity := items[i].Diagnostic.Severity
			jSeverity := items[j].Diagnostic.Severity
			if iSeverity != jSeverity {
				return iSeverity < jSeverity // Lower severity number = higher priority
			}
			return items[i].FilePath < items[j].FilePath
		})
		
		if len(items) > 0 {
			lspDiagnostics[lspName] = items
		}
	}
	
	// Add sections for each LSP with diagnostics
	for lspName, items := range lspDiagnostics {
		// Add section header
		diagnosticItems = append(diagnosticItems, commands.NewItemSection(lspName))
		
		// Add diagnostic items
		for _, item := range items {
			title := d.formatDiagnosticTitle(item)
			diagnosticItems = append(diagnosticItems, completions.NewCompletionItem(title, item))
		}
	}
	
	d.diagnosticsList.SetItems(diagnosticItems)
}

func (d *diagnosticsDialogCmp) formatDiagnosticTitle(item DiagnosticItem) string {
	severity := "Info"
	switch item.Diagnostic.Severity {
	case protocol.SeverityError:
		severity = "Error"
	case protocol.SeverityWarning:
		severity = "Warn"
	case protocol.SeverityHint:
		severity = "Hint"
	}
	
	// Extract filename from path
	parts := strings.Split(item.FilePath, "/")
	filename := parts[len(parts)-1]
	
	location := fmt.Sprintf("%s:%d:%d", 
		filename, 
		item.Diagnostic.Range.Start.Line+1, 
		item.Diagnostic.Range.Start.Character+1)
	
	// Truncate message if too long
	message := item.Diagnostic.Message
	if len(message) > 60 {
		message = message[:57] + "..."
	}
	
	return fmt.Sprintf("[%s] %s - %s", severity, location, message)
}