package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/pkg/extensions"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pane constants
const (
	extensionListPane = iota
	selectedPane
)

// ExtensionSelectorModel handles the split-pane extension selection interface
type ExtensionSelectorModel struct {
	extManager    *extensions.Manager
	configManager *config.ConfigManager

	// Panes
	activePane    int
	extensionList list.Model
	selectedList  list.Model
	nameInput     textinput.Model
	descInput     textinput.Model

	// State
	allExtensions  []extensions.Extension
	selected       map[int]struct{} // Track selected by original index
	extensionItems []list.Item
	selectedItems  []list.Item

	// Input state
	ShowNameInput bool // Export for main model access
	ShowDescInput bool // Export for main model access

	// Result
	configName    string
	description   string
	selectedExts  []string
	editingConfig *config.SavedConfig
	finished      bool
	cancelled     bool

	// Dimensions
	width  int
	height int
}

// ExtensionItem implements list.Item for the extension list
type ExtensionItem struct {
	ext      extensions.Extension
	selected bool
}

func (i ExtensionItem) FilterValue() string { return i.ext.Name }
func (i ExtensionItem) Title() string {
	title := i.ext.Name
	if i.selected {
		title = "✓ " + title
	} else {
		title = "  " + title
	}
	return title
}
func (i ExtensionItem) Description() string {
	return fmt.Sprintf("%s (%s)", i.ext.Description, i.ext.Kind)
}

// SelectedItem implements list.Item for the selected extensions list
type SelectedItem struct {
	name string
}

func (i SelectedItem) FilterValue() string { return i.name }
func (i SelectedItem) Title() string       { return "✓ " + i.name }
func (i SelectedItem) Description() string { return "" }

// NewExtensionSelectorModel creates a new extension selector
func NewExtensionSelectorModel(extManager *extensions.Manager, editConfig *config.SavedConfig) *ExtensionSelectorModel {
	model := &ExtensionSelectorModel{
		extManager:    extManager,
		configManager: config.NewConfigManager(),
		activePane:    extensionListPane,
		selected:      make(map[int]struct{}),
		editingConfig: editConfig,
		width:         80,
		height:        24,
	}

	model.setupInputs()
	model.loadExtensions()
	model.setupLists()

	// If editing, pre-populate selections
	if editConfig != nil {
		model.populateFromConfig(editConfig)
	}

	return model
}

func (m *ExtensionSelectorModel) setupInputs() {
	// Name input
	m.nameInput = textinput.New()
	m.nameInput.Placeholder = "Enter configuration name (e.g., 'my-app-stack')"
	m.nameInput.Width = 50

	// Description input
	m.descInput = textinput.New()
	m.descInput.Placeholder = "Optional: Brief description of this configuration"
	m.descInput.Width = 50

	// Pre-populate if editing
	if m.editingConfig != nil {
		m.nameInput.SetValue(m.editingConfig.Name)
		m.descInput.SetValue(m.editingConfig.Description)
		m.configName = m.editingConfig.Name
		m.description = m.editingConfig.Description
	}
}

func (m *ExtensionSelectorModel) loadExtensions() {
	exts, err := m.extManager.GetAllExtensions()
	if err != nil {
		m.allExtensions = []extensions.Extension{}
	} else {
		m.allExtensions = exts
	}

	// Sort extensions alphabetically
	sort.Slice(m.allExtensions, func(i, j int) bool {
		return m.allExtensions[i].Name < m.allExtensions[j].Name
	})
}

func (m *ExtensionSelectorModel) setupLists() {
	// Extension list (left pane)
	m.extensionItems = make([]list.Item, len(m.allExtensions))
	for i, ext := range m.allExtensions {
		_, selected := m.selected[i]
		m.extensionItems[i] = ExtensionItem{ext: ext, selected: selected}
	}

	m.extensionList = list.New(m.extensionItems, list.NewDefaultDelegate(), 0, 0)
	m.extensionList.Title = "Available Extensions"
	m.extensionList.SetShowStatusBar(false)
	m.extensionList.SetFilteringEnabled(true)
	m.extensionList.SetShowHelp(false)

	// Selected extensions list (right pane) - initialize first
	m.selectedItems = []list.Item{} // Initialize empty slice
	m.selectedList = list.New(m.selectedItems, list.NewDefaultDelegate(), 0, 0)
	m.selectedList.Title = "Selected Extensions (0)"
	m.selectedList.SetShowStatusBar(false)
	m.selectedList.SetFilteringEnabled(false)
	m.selectedList.SetShowHelp(false)

	// Now update the selected list with actual selections
	m.updateSelectedList()
}

func (m *ExtensionSelectorModel) populateFromConfig(cfg *config.SavedConfig) {
	// Find and select extensions from the config
	for _, extName := range cfg.Extensions {
		for i, ext := range m.allExtensions {
			if ext.Name == extName {
				m.selected[i] = struct{}{}
				break
			}
		}
	}
	m.updateLists()
}

func (m *ExtensionSelectorModel) updateLists() {
	// Update extension list items with selection status
	for i, ext := range m.allExtensions {
		_, selected := m.selected[i]
		m.extensionItems[i] = ExtensionItem{ext: ext, selected: selected}
	}
	m.extensionList.SetItems(m.extensionItems)

	// Update selected list
	m.updateSelectedList()
}

func (m *ExtensionSelectorModel) updateSelectedList() {
	m.selectedItems = []list.Item{}
	selectedNames := []string{}

	for i := range m.selected {
		if i < len(m.allExtensions) {
			selectedNames = append(selectedNames, m.allExtensions[i].Name)
		}
	}

	// Sort selected items
	sort.Strings(selectedNames)

	for _, name := range selectedNames {
		m.selectedItems = append(m.selectedItems, SelectedItem{name: name})
	}

	m.selectedList.SetItems(m.selectedItems)
	m.selectedList.Title = fmt.Sprintf("Selected Extensions (%d)", len(m.selectedItems))
}

func (m *ExtensionSelectorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ExtensionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions with proper sizing
		listWidth := max(30, (m.width-8)/2) // Minimum 30 chars, split remaining
		listHeight := max(10, m.height-8)   // Minimum 10 lines, leave room for help

		m.extensionList.SetWidth(listWidth)
		m.extensionList.SetHeight(listHeight)
		m.selectedList.SetWidth(listWidth)
		m.selectedList.SetHeight(listHeight)

	case tea.KeyMsg:
		// Handle name input
		if m.ShowNameInput {
			switch msg.String() {
			case "enter":
				m.configName = m.nameInput.Value()
				m.ShowNameInput = false
				if m.configName == "" {
					m.configName = "unnamed_config"
				}

				// If we have selections, show description input
				if len(m.selected) > 0 {
					m.ShowDescInput = true
					m.descInput.Focus()
				}
				return m, nil

			case "esc":
				m.ShowNameInput = false
				return m, nil
			}

			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd
		}

		// Handle description input
		if m.ShowDescInput {
			switch msg.String() {
			case "enter":
				m.description = m.descInput.Value()
				m.ShowDescInput = false
				m.finishSelection()
				return m, tea.Quit

			case "esc":
				m.ShowDescInput = false
				m.finishSelection()
				return m, tea.Quit
			}

			m.descInput, cmd = m.descInput.Update(msg)
			return m, cmd
		}

		// Regular key handling
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			m.finished = true
			return m, tea.Quit

		case "tab":
			// Switch between panes
			m.activePane = (m.activePane + 1) % 2

		case "enter":
			// Confirm selection and ask for name
			if len(m.selected) == 0 {
				return m, nil // No selection made
			}

			// If editing, skip name input
			if m.editingConfig != nil {
				m.configName = m.editingConfig.Name
				m.description = m.editingConfig.Description
				m.finishSelection()
				return m, tea.Quit
			}

			// Show name input
			m.ShowNameInput = true
			m.nameInput.Focus()
			return m, nil

		case "space":
			// Toggle selection (only in extension list pane)
			if m.activePane == extensionListPane {
				if selectedItem, ok := m.extensionList.SelectedItem().(ExtensionItem); ok {
					// Find the original index
					for i, ext := range m.allExtensions {
						if ext.Name == selectedItem.ext.Name {
							if _, exists := m.selected[i]; exists {
								delete(m.selected, i)
							} else {
								m.selected[i] = struct{}{}
							}
							break
						}
					}
					m.updateLists()
				}
				return m, tea.Batch(cmds...) // Return early to prevent list from processing space key
			}

		case "d", "x":
			// Remove from selection (only in selected pane)
			if m.activePane == selectedPane {
				if selectedItem, ok := m.selectedList.SelectedItem().(SelectedItem); ok {
					// Find and remove from selected map
					for i, ext := range m.allExtensions {
						if ext.Name == selectedItem.name {
							delete(m.selected, i)
							break
						}
					}
					m.updateLists()
				}
				return m, tea.Batch(cmds...) // Return early to prevent list from processing d/x keys
			}
		}
	}

	// Update the appropriate list based on active pane (only if key wasn't handled above)
	switch m.activePane {
	case extensionListPane:
		m.extensionList, cmd = m.extensionList.Update(msg)
		cmds = append(cmds, cmd)
	case selectedPane:
		m.selectedList, cmd = m.selectedList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ExtensionSelectorModel) View() string {
	if m.ShowNameInput {
		return m.renderNameInput()
	}

	if m.ShowDescInput {
		return m.renderDescInput()
	}

	return m.renderSplitPane()
}

func (m *ExtensionSelectorModel) renderNameInput() string {
	content := lipgloss.NewStyle().
		Padding(4, 2).
		Align(lipgloss.Center).
		Render(
			"💾 Save Configuration\n\n" +
				m.nameInput.View() + "\n\n" +
				"Press [Enter] to continue or [Esc] to cancel",
		)
	return content
}

func (m *ExtensionSelectorModel) renderDescInput() string {
	content := lipgloss.NewStyle().
		Padding(4, 2).
		Align(lipgloss.Center).
		Render(
			"📝 Configuration Description\n\n" +
				m.descInput.View() + "\n\n" +
				"Press [Enter] to save or [Esc] to skip",
		)
	return content
}

func (m *ExtensionSelectorModel) renderSplitPane() string {
	// Calculate pane dimensions
	paneWidth := max(30, (m.width-6)/2)

	leftPaneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(paneWidth).
		Height(m.height - 6)

	rightPaneStyle := leftPaneStyle

	// Highlight active pane
	if m.activePane == extensionListPane {
		leftPaneStyle = leftPaneStyle.BorderForeground(lipgloss.Color("12"))
	} else {
		rightPaneStyle = rightPaneStyle.BorderForeground(lipgloss.Color("12"))
	}

	leftPane := leftPaneStyle.Render(m.extensionList.View())
	rightPane := rightPaneStyle.Render(m.selectedList.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Add help text
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, content, help)
}

func (m *ExtensionSelectorModel) renderHelp() string {
	helpText := []string{
		"[Tab] Switch panes",
		"[Space] Toggle selection",
	}

	if m.activePane == selectedPane {
		helpText = append(helpText, "[d/x] Remove from selection")
	}

	helpText = append(helpText,
		"[Enter] Confirm",
		"[q] Quit",
	)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(strings.Join(helpText, " • "))
}

func (m *ExtensionSelectorModel) finishSelection() {
	// Build final extension list
	for i := range m.selected {
		if i < len(m.allExtensions) {
			m.selectedExts = append(m.selectedExts, m.allExtensions[i].Name)
		}
	}

	sort.Strings(m.selectedExts)
	m.finished = true
}

// GetResult returns the selected extensions and configuration details
func (m *ExtensionSelectorModel) GetResult() ([]string, string, string, bool) {
	if m.cancelled {
		return nil, "", "", true // cancelled, so return empty results but finished=true
	}
	return m.selectedExts, m.configName, m.description, m.finished
}

// Helper function for max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
