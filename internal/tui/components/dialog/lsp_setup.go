package dialog

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/lsp/protocol"
	"github.com/opencode-ai/opencode/internal/lsp/setup"
	utilComponents "github.com/opencode-ai/opencode/internal/tui/components/util"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

// LSPSetupStep represents the current step in the LSP setup wizard
type LSPSetupStep int

const (
	StepIntroduction LSPSetupStep = iota
	StepLanguageSelection
	StepConfirmation
	StepInstallation
)

// LSPSetupWizard is a component that guides users through LSP setup
type LSPSetupWizard struct {
	ctx            context.Context
	step           LSPSetupStep
	width, height  int
	languages      map[protocol.LanguageKind]int
	selectedLangs  map[protocol.LanguageKind]bool
	availableLSPs  setup.LSPServerMap
	selectedLSPs   map[protocol.LanguageKind]setup.LSPServerInfo
	installResults map[protocol.LanguageKind]setup.InstallationResult
	isMonorepo     bool
	projectDirs    []string
	langList       utilComponents.SimpleList[LSPItem]
	serverList     utilComponents.SimpleList[LSPItem]
	spinner        spinner.Model
	installing     bool
	currentInstall string
	installOutput  []string // Store installation output
	keys           lspSetupKeyMap
	error          string
	program        *tea.Program
}

// LSPItem represents an item in the language or server list
type LSPItem struct {
	title       string
	description string
	selected    bool
	language    protocol.LanguageKind
	server      setup.LSPServerInfo
}

// Render implements SimpleListItem interface
func (i LSPItem) Render(selected bool, width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	descStyle := baseStyle.Width(width).Foreground(t.TextMuted())
	itemStyle := baseStyle.Width(width).
		Foreground(t.Text()).
		Background(t.Background())

	if selected {
		itemStyle = itemStyle.
			Background(t.Primary()).
			Foreground(t.Background()).
			Bold(true)
		descStyle = descStyle.
			Background(t.Primary()).
			Foreground(t.Background())
	}

	title := i.title
	if i.selected {
		title = "[x] " + i.title
	} else {
		title = "[ ] " + i.title
	}

	titleStr := itemStyle.Padding(0, 1).Render(title)
	if i.description != "" {
		description := descStyle.Padding(0, 1).Render(i.description)
		return lipgloss.JoinVertical(lipgloss.Left, titleStr, description)
	}
	return titleStr
}

// NewLSPSetupWizard creates a new LSPSetupWizard
func NewLSPSetupWizard(ctx context.Context) *LSPSetupWizard {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &LSPSetupWizard{
		ctx:            ctx,
		step:           StepIntroduction,
		selectedLangs:  make(map[protocol.LanguageKind]bool),
		selectedLSPs:   make(map[protocol.LanguageKind]setup.LSPServerInfo),
		installResults: make(map[protocol.LanguageKind]setup.InstallationResult),
		installOutput:  make([]string, 0, 10), // Initialize with capacity for 10 lines
		spinner:        s,
		keys:           DefaultLSPSetupKeyMap(),
	}
}

type lspSetupKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Next   key.Binding
	Back   key.Binding
	Quit   key.Binding
}

// DefaultLSPSetupKeyMap returns the default key bindings for the LSP setup wizard
func DefaultLSPSetupKeyMap() lspSetupKeyMap {
	return lspSetupKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "select"),
		),
		Next: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "next"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/quit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
	}
}

// ShortHelp implements key.Map
func (k lspSetupKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
		k.Select,
		k.Next,
		k.Back,
	}
}

// FullHelp implements key.Map
func (k lspSetupKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// Init implements tea.Model
func (m *LSPSetupWizard) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.detectLanguages,
	)
}

// detectLanguages is a command that detects languages in the workspace
func (m *LSPSetupWizard) detectLanguages() tea.Msg {
	languages, err := setup.DetectProjectLanguages(config.WorkingDirectory())
	if err != nil {
		return lspSetupErrorMsg{err: err}
	}

	isMonorepo, projectDirs := setup.DetectMonorepo(config.WorkingDirectory())

	primaryLangs := setup.GetPrimaryLanguages(languages, 10)

	availableLSPs := setup.DiscoverInstalledLSPs()

	recommendedLSPs := setup.GetRecommendedLSPServers(primaryLangs)
	for lang, servers := range recommendedLSPs {
		if _, ok := availableLSPs[lang]; !ok {
			availableLSPs[lang] = servers
		}
	}

	return lspSetupDetectMsg{
		languages:     languages,
		primaryLangs:  primaryLangs,
		availableLSPs: availableLSPs,
		isMonorepo:    isMonorepo,
		projectDirs:   projectDirs,
	}
}

// Update implements tea.Model
func (m *LSPSetupWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle space key directly for language selection
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " && m.step == StepLanguageSelection {
		item, idx := m.langList.GetSelectedItem()
		if idx != -1 {
			m.selectedLangs[item.language] = !m.selectedLangs[item.language]
			return m, m.updateLanguageSelection()
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, util.CmdHandler(CloseLSPSetupMsg{Configure: false})
		case key.Matches(msg, m.keys.Back):
			if m.step > StepIntroduction {
				m.step--
				return m, nil
			}
			return m, util.CmdHandler(CloseLSPSetupMsg{Configure: false})
		case key.Matches(msg, m.keys.Next):
			return m.handleEnter()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions
		if m.langList != nil {
			m.langList.SetMaxWidth(min(80, m.width-10))
		}
		if m.serverList != nil {
			m.serverList.SetMaxWidth(min(80, m.width-10))
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.installing {
			// Only continue ticking if we're still installing
			cmds = append(cmds, cmd)
		}

	case lspSetupDetectMsg:
		m.languages = msg.languages
		m.availableLSPs = msg.availableLSPs
		m.isMonorepo = msg.isMonorepo
		m.projectDirs = msg.projectDirs

		// Create language list items - only for languages with available servers
		items := []LSPItem{}
		for _, lang := range msg.primaryLangs {
			// Check if servers are available for this language
			hasServers := false
			if servers, ok := m.availableLSPs[lang.Language]; ok && len(servers) > 0 {
				hasServers = true
			}

			// Only add languages that have servers available
			if hasServers {
				item := LSPItem{
					title:    string(lang.Language),
					selected: false,
					language: lang.Language,
				}

				items = append(items, item)

				// Pre-select languages with high scores
				if lang.Score > 10 {
					m.selectedLangs[lang.Language] = true
				}
			}
		}

		// Create the language list
		m.langList = utilComponents.NewSimpleList(items, 10, "No languages with available servers detected", true)

		// Move to the next step
		m.step = StepLanguageSelection

		// Update the selection status in the list
		return m, m.updateLanguageSelection()

	case lspSetupErrorMsg:
		m.error = msg.err.Error()
		return m, nil

	case lspSetupInstallMsg:
		m.installResults[msg.language] = msg.result

		// Add output from the installation result
		if msg.output != "" {
			m.addOutputLine(msg.output)
		}

		// Add success/failure message with clear formatting
		if msg.result.Success {
			m.addOutputLine(fmt.Sprintf("✓ Successfully installed %s for %s", msg.result.ServerName, msg.language))
		} else {
			m.addOutputLine(fmt.Sprintf("✗ Failed to install %s for %s", msg.result.ServerName, msg.language))
		}

		m.installing = false

		if len(m.installResults) == len(m.selectedLSPs) {
			// All installations are complete, move to the summary step
			m.step = StepInstallation
		} else {
			// Continue with the next installation
			return m, m.installNextServer()
		}
	}

	// Handle list updates
	if m.step == StepLanguageSelection {
		u, cmd := m.langList.Update(msg)
		m.langList = u.(utilComponents.SimpleList[LSPItem])
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleEnter handles the enter key press based on the current step
func (m *LSPSetupWizard) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepIntroduction: // Introduction
		return m, m.detectLanguages

	case StepLanguageSelection: // Language selection
		// Check if any languages are selected
		hasSelected := false

		// Create a sorted list of languages for consistent ordering
		var selectedLangs []protocol.LanguageKind
		for lang, selected := range m.selectedLangs {
			if selected {
				selectedLangs = append(selectedLangs, lang)
				hasSelected = true
			}
		}

		// Sort languages alphabetically for consistent display
		sort.Slice(selectedLangs, func(i, j int) bool {
			return string(selectedLangs[i]) < string(selectedLangs[j])
		})

		// Auto-select servers for each language
		for _, lang := range selectedLangs {
			// Auto-select the recommended or first server for each language
			if servers, ok := m.availableLSPs[lang]; ok && len(servers) > 0 {
				// First try to find a recommended server
				foundRecommended := false
				for _, server := range servers {
					if server.Recommended {
						m.selectedLSPs[lang] = server
						foundRecommended = true
						break
					}
				}

				// If no recommended server, use the first one
				if !foundRecommended && len(servers) > 0 {
					m.selectedLSPs[lang] = servers[0]
				}
			} else {
				// No servers available for this language, deselect it
				m.selectedLangs[lang] = false
				// Update the UI to reflect this change
				return m, m.updateLanguageSelection()
			}
		}

		if !hasSelected {
			// No language selected, show error
			m.error = "Please select at least one language"
			return m, nil
		}

		// Skip server selection and go directly to confirmation
		m.step = StepConfirmation
		return m, nil

	case StepConfirmation: // Confirmation
		// Start installation
		m.step = StepInstallation
		m.installing = true
		// Start the spinner and begin installation
		return m, tea.Batch(
			m.spinner.Tick,
			m.installNextServer(),
		)

	case StepInstallation: // Summary
		// Save configuration and close
		return m, util.CmdHandler(CloseLSPSetupMsg{
			Configure: true,
			Servers:   m.selectedLSPs,
		})
	}

	return m, nil
}

// View implements tea.Model
func (m *LSPSetupWizard) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	// Calculate width needed for content
	maxWidth := min(80, m.width-10)

	title := baseStyle.
		Foreground(t.Primary()).
		Bold(true).
		Width(maxWidth).
		Padding(0, 1).
		Render("LSP Setup Wizard")

	var content string

	switch m.step {
	case StepIntroduction: // Introduction
		content = m.renderIntroduction(baseStyle, t, maxWidth)
	case StepLanguageSelection: // Language selection
		content = m.renderLanguageSelection(baseStyle, t, maxWidth)
	case StepConfirmation: // Confirmation
		content = m.renderConfirmation(baseStyle, t, maxWidth)
	case StepInstallation: // Installation/Summary
		content = m.renderInstallation(baseStyle, t, maxWidth)
	}

	// Add error message if any
	if m.error != "" {
		errorMsg := baseStyle.
			Foreground(t.Error()).
			Width(maxWidth).
			Padding(1, 1).
			Render("Error: " + m.error)

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			content,
			errorMsg,
		)
	}

	// Add help text
	helpText := baseStyle.
		Foreground(t.TextMuted()).
		Width(maxWidth).
		Padding(1, 1).
		Render(m.getHelpText())

	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		baseStyle.Width(maxWidth).Render(""),
		content,
		helpText,
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderNormal()).
		Width(lipgloss.Width(fullContent) + 4).
		Render(fullContent)
}

// renderIntroduction renders the introduction step
func (m *LSPSetupWizard) renderIntroduction(baseStyle lipgloss.Style, t theme.Theme, maxWidth int) string {
	explanation := baseStyle.
		Foreground(t.Text()).
		Width(maxWidth).
		Padding(0, 1).
		Render("OpenCode can automatically configure Language Server Protocol (LSP) integration for your project. LSP provides code intelligence features like error checking, diagnostics, and more.")

	if m.languages == nil {
		// Show spinner while detecting languages
		spinner := baseStyle.
			Foreground(t.Primary()).
			Width(maxWidth).
			Padding(1, 1).
			Render(m.spinner.View() + " Detecting languages in your project...")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			explanation,
			spinner,
		)
	}

	nextPrompt := baseStyle.
		Foreground(t.Primary()).
		Width(maxWidth).
		Padding(1, 1).
		Render("Press Enter to continue")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		explanation,
		nextPrompt,
	)
}

// renderLanguageSelection renders the language selection step
func (m *LSPSetupWizard) renderLanguageSelection(baseStyle lipgloss.Style, t theme.Theme, maxWidth int) string {
	explanation := baseStyle.
		Foreground(t.Text()).
		Width(maxWidth).
		Padding(0, 1).
		Render("Select the languages you want to configure LSP for. Only languages with available servers are shown. Use Space to toggle selection, Enter to continue.")

	// Show monorepo info if detected
	monorepoInfo := ""
	if m.isMonorepo {
		monorepoInfo = baseStyle.
			Foreground(t.TextMuted()).
			Width(maxWidth).
			Padding(0, 1).
			Render(fmt.Sprintf("Monorepo detected with %d projects", len(m.projectDirs)))
	}

	// Set max width for the list
	m.langList.SetMaxWidth(maxWidth)

	// Render the language list
	listView := m.langList.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		explanation,
		monorepoInfo,
		listView,
	)
}

// renderConfirmation renders the confirmation step
func (m *LSPSetupWizard) renderConfirmation(baseStyle lipgloss.Style, t theme.Theme, maxWidth int) string {
	explanation := baseStyle.
		Foreground(t.Text()).
		Width(maxWidth).
		Padding(0, 1).
		Render("Review your LSP configuration. Press Enter to install missing servers and save the configuration.")

	// Get languages in a sorted order for consistent display
	var languages []protocol.LanguageKind
	for lang := range m.selectedLSPs {
		languages = append(languages, lang)
	}

	// Sort languages alphabetically
	sort.Slice(languages, func(i, j int) bool {
		return string(languages[i]) < string(languages[j])
	})

	// Build the configuration summary
	var configLines []string
	for _, lang := range languages {
		server := m.selectedLSPs[lang]
		configLines = append(configLines, fmt.Sprintf("%s: %s", lang, server.Name))
	}

	configSummary := baseStyle.
		Foreground(t.Text()).
		Width(maxWidth).
		Padding(1, 1).
		Render(strings.Join(configLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		explanation,
		configSummary,
	)
}

// renderInstallation renders the installation/summary step
func (m *LSPSetupWizard) renderInstallation(baseStyle lipgloss.Style, t theme.Theme, maxWidth int) string {
	if m.installing {
		// Show installation progress with proper styling
		spinnerStyle := baseStyle.
			Foreground(t.Primary()).
			Background(t.Background()).
			Width(maxWidth).
			Padding(1, 1)

		spinnerText := m.spinner.View() + " Installing " + m.currentInstall + "..."

		// Show output if available
		var content string
		if len(m.installOutput) > 0 {
			outputStyle := baseStyle.
				Foreground(t.TextMuted()).
				Background(t.Background()).
				Width(maxWidth).
				Padding(1, 1)

			outputText := strings.Join(m.installOutput, "\n")
			outputContent := outputStyle.Render(outputText)

			content = lipgloss.JoinVertical(
				lipgloss.Left,
				spinnerStyle.Render(spinnerText),
				outputContent,
			)
		} else {
			content = spinnerStyle.Render(spinnerText)
		}

		return content
	}

	// Show installation results
	explanation := baseStyle.
		Foreground(t.Text()).
		Width(maxWidth).
		Padding(0, 1).
		Render("LSP server installation complete. Press Enter to save the configuration and exit.")

	// Build the installation summary
	var resultLines []string
	for lang, result := range m.installResults {
		status := "✓"
		statusColor := t.Success()
		if !result.Success {
			status = "✗"
			statusColor = t.Error()
		}

		line := fmt.Sprintf("%s %s: %s",
			baseStyle.Foreground(statusColor).Render(status),
			lang,
			result.ServerName)

		resultLines = append(resultLines, line)
	}

	// Style the result summary with a header
	resultHeader := baseStyle.
		Foreground(t.Primary()).
		Bold(true).
		Width(maxWidth).
		Padding(1, 1).
		Render("Installation Results:")

	resultSummary := baseStyle.
		Width(maxWidth).
		Padding(0, 2). // Indent the results
		Render(strings.Join(resultLines, "\n"))

	// Show output if available
	var content string
	if len(m.installOutput) > 0 {
		// Create a header for the output section
		outputHeader := baseStyle.
			Foreground(t.Primary()).
			Bold(true).
			Width(maxWidth).
			Padding(1, 1).
			Render("Installation Output:")

		// Style the output
		outputStyle := baseStyle.
			Foreground(t.TextMuted()).
			Background(t.Background()).
			Width(maxWidth).
			Padding(0, 2) // Indent the output

		outputText := strings.Join(m.installOutput, "\n")
		outputContent := outputStyle.Render(outputText)

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			explanation,
			baseStyle.Render(""), // Add a blank line for spacing
			resultHeader,
			resultSummary,
			baseStyle.Render(""), // Add a blank line for spacing
			outputHeader,
			outputContent,
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			explanation,
			baseStyle.Render(""), // Add a blank line for spacing
			resultHeader,
			resultSummary,
		)
	}

	return content
}

// getHelpText returns the help text for the current step
func (m *LSPSetupWizard) getHelpText() string {
	switch m.step {
	case StepIntroduction:
		return "Enter: Continue • Esc: Quit"
	case StepLanguageSelection:
		return "↑/↓: Navigate • Space: Toggle selection • Enter: Continue • Esc: Quit"
	case StepConfirmation:
		return "Enter: Install and configure • Esc: Back"
	case StepInstallation:
		if m.installing {
			return "Installing LSP servers..."
		}
		return "Enter: Save and exit • Esc: Back"
	}

	return ""
}

// updateLanguageSelection updates the selection status in the language list
func (m *LSPSetupWizard) updateLanguageSelection() tea.Cmd {
	return func() tea.Msg {
		items := m.langList.GetItems()
		updatedItems := make([]LSPItem, 0, len(items))

		for _, item := range items {
			// Only update the selected state, preserve the item otherwise
			newItem := item
			newItem.selected = m.selectedLangs[item.language]
			updatedItems = append(updatedItems, newItem)
		}

		// Set the items - the selected index will be preserved by the SimpleList implementation
		m.langList.SetItems(updatedItems)

		return nil
	}
}

// createServerListForLanguage creates the server list for a specific language
func (m *LSPSetupWizard) createServerListForLanguage(lang protocol.LanguageKind) tea.Cmd {
	return func() tea.Msg {
		items := []LSPItem{}

		if servers, ok := m.availableLSPs[lang]; ok {
			for _, server := range servers {
				description := server.Description
				if server.Recommended {
					description += " (Recommended)"
				}

				items = append(items, LSPItem{
					title:       server.Name,
					description: description,
					language:    lang,
					server:      server,
				})
			}
		}

		// If no servers available, add a placeholder
		if len(items) == 0 {
			items = append(items, LSPItem{
				title:       "No LSP servers available for " + string(lang),
				description: "You'll need to install a server manually",
				language:    lang,
			})
		}

		// Create the server list
		m.serverList = utilComponents.NewSimpleList(items, 10, "No servers available", true)

		// Move to the server selection step
		m.step = 2

		return nil
	}
}

// getCurrentLanguage returns the language currently being configured
func (m *LSPSetupWizard) getCurrentLanguage() protocol.LanguageKind {
	items := m.serverList.GetItems()
	if len(items) == 0 {
		return ""
	}
	return items[0].language
}

// getNextLanguage returns the next language to configure after the current one
func (m *LSPSetupWizard) getNextLanguage(currentLang protocol.LanguageKind) protocol.LanguageKind {
	foundCurrent := false

	for lang := range m.selectedLangs {
		if m.selectedLangs[lang] {
			if foundCurrent {
				return lang
			}

			if lang == currentLang {
				foundCurrent = true
			}
		}
	}

	return ""
}

// installNextServer installs the next server in the queue
func (m *LSPSetupWizard) installNextServer() tea.Cmd {
	return func() tea.Msg {
		for lang, server := range m.selectedLSPs {
			if _, ok := m.installResults[lang]; !ok {
				if _, err := exec.LookPath(server.Command); err == nil {
					// Server is already installed
					output := fmt.Sprintf("%s is already installed", server.Name)
					m.installResults[lang] = setup.InstallationResult{
						ServerName: server.Name,
						Success:    true,
						Output:     output,
					}

					// Add output line
					m.addOutputLine(output)

					// Continue with next server immediately
					return m.installNextServer()()
				}

				// Install this server
				m.installing = true
				m.currentInstall = fmt.Sprintf("%s for %s", server.Name, lang)

				// Add initial output line
				m.addOutputLine(fmt.Sprintf("Installing %s for %s...", server.Name, lang))

				// Create a channel to receive the installation result
				resultCh := make(chan setup.InstallationResult)

				go func(l protocol.LanguageKind, s setup.LSPServerInfo) {
					result := setup.InstallLSPServer(m.ctx, s)
					resultCh <- result
				}(lang, server)

				// Return a command that will wait for the installation to complete
				// and also keep the spinner updating
				return tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						result := <-resultCh
						return lspSetupInstallMsg{
							language: lang,
							result:   result,
							output:   result.Output,
						}
					},
				)
			}
		}

		// All servers have been installed
		m.installing = false
		m.step = StepInstallation
		return nil
	}
}

// SetSize sets the size of the component
func (m *LSPSetupWizard) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Update list max width if lists are initialized
	if m.langList != nil {
		m.langList.SetMaxWidth(min(80, width-10))
	}
	if m.serverList != nil {
		m.serverList.SetMaxWidth(min(80, width-10))
	}
}

// addOutputLine adds a line to the installation output, keeping only the last 10 lines
func (m *LSPSetupWizard) addOutputLine(line string) {
	// Split the line into multiple lines if it contains newlines
	lines := strings.Split(line, "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}

		// Add the line to the output
		m.installOutput = append(m.installOutput, l)

		// Keep only the last 10 lines
		if len(m.installOutput) > 10 {
			m.installOutput = m.installOutput[len(m.installOutput)-10:]
		}
	}
}

// Bindings implements layout.Bindings
func (m *LSPSetupWizard) Bindings() []key.Binding {
	return m.keys.ShortHelp()
}

// Message types for the LSP setup wizard
type lspSetupDetectMsg struct {
	languages     map[protocol.LanguageKind]int
	primaryLangs  []setup.LanguageScore
	availableLSPs setup.LSPServerMap
	isMonorepo    bool
	projectDirs   []string
}

type lspSetupErrorMsg struct {
	err error
}

type lspSetupInstallMsg struct {
	language protocol.LanguageKind
	result   setup.InstallationResult
	output   string // Installation output
}

// CloseLSPSetupMsg is a message that is sent when the LSP setup wizard is closed
type CloseLSPSetupMsg struct {
	Configure bool
	Servers   map[protocol.LanguageKind]setup.LSPServerInfo
}

// ShowLSPSetupMsg is a message that is sent to show the LSP setup wizard
type ShowLSPSetupMsg struct {
	Show bool
}
