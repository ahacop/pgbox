package ui

import (
	"fmt"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/pkg/extensions"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab constants
const (
	configsTab = iota
	newConfigTab
)

var (
	// Tab styles
	inactiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "│",
		BottomRight: "│",
	}

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	inactiveTabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(highlightColor).
				Padding(0, 1)

	activeTabStyle = inactiveTabStyle.
			Border(activeTabBorder, true)

	windowStyle = lipgloss.NewStyle().
			BorderForeground(highlightColor).
			Padding(2, 0).
			Align(lipgloss.Center).
			Border(lipgloss.NormalBorder()).
			UnsetBorderTop()
)

// MainModel represents the main tabbed interface
type MainModel struct {
	activeTab     int
	configManager *config.ConfigManager
	extManager    *extensions.Manager

	// Tab content models
	configListModel *ConfigListModel
	newConfigModel  *ExtensionSelectorModel

	// Window sizing
	windowWidth  int
	windowHeight int

	// Result to return to caller
	selectedConfig     *config.SavedConfig
	selectedExtensions []string
	finished           bool
	cancelled          bool
}

// NewMainModel creates a new main model
func NewMainModel(extManager *extensions.Manager) *MainModel {
	configManager := config.NewConfigManager()
	return &MainModel{
		activeTab:       configsTab,
		configManager:   configManager,
		extManager:      extManager,
		configListModel: NewConfigListModel(configManager),
		newConfigModel:  NewExtensionSelectorModel(extManager, nil),
		windowWidth:     80,
		windowHeight:    24,
	}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			m.finished = true
			return m, tea.Quit

		case "left", "right", "tab":
			// Only allow tab switching if not in the middle of an operation
			if !m.isInSubProcess() {
				if msg.String() == "right" || msg.String() == "tab" {
					m.activeTab = (m.activeTab + 1) % 2
				} else {
					m.activeTab = (m.activeTab - 1 + 2) % 2
				}
			}
		}
	}

	// Update the active tab model
	switch m.activeTab {
	case configsTab:
		result, cmd := m.configListModel.Update(msg)
		if updatedModel, ok := result.(*ConfigListModel); ok {
			*m.configListModel = *updatedModel
		}
		_ = cmd

		// Check if config list wants to finish
		selectedConfig, editConfig, finished := m.configListModel.GetResult()
		if finished {
			if selectedConfig != nil {
				// Launch selected configuration
				m.selectedConfig = selectedConfig
				m.finished = true
				return m, tea.Quit
			} else if editConfig != nil {
				// Switch to edit mode in new config tab
				m.newConfigModel = NewExtensionSelectorModel(m.extManager, editConfig)
				m.activeTab = newConfigTab
			}
		}

	case newConfigTab:
		result, cmd := m.newConfigModel.Update(msg)
		if updatedModel, ok := result.(*ExtensionSelectorModel); ok {
			*m.newConfigModel = *updatedModel
		}
		_ = cmd

		// Check if extension selector wants to finish
		extensions, configName, description, finished := m.newConfigModel.GetResult()
		if finished && len(extensions) > 0 {
			m.selectedExtensions = extensions

			// Save the configuration if it's named
			if configName != "" {
				currentConfig := config.NewConfig()
				savedConfig := config.CreateConfigFromCurrent(currentConfig, extensions, configName, description)
				_ = m.configManager.SaveConfig(savedConfig)
			}

			m.finished = true
			return m, tea.Quit
		}
	}

	return m, cmd
}

// isInSubProcess checks if we're in the middle of a sub-process that shouldn't allow tab switching
func (m MainModel) isInSubProcess() bool {
	// Check if config list is showing delete confirmation
	if m.activeTab == configsTab && m.configListModel.ShowConfirmDelete {
		return true
	}

	// Check if extension selector is showing input dialogs
	if m.activeTab == newConfigTab && (m.newConfigModel.ShowNameInput || m.newConfigModel.ShowDescInput) {
		return true
	}

	return false
}

func (m MainModel) View() string {
	// Create tabs
	var renderedTabs []string

	// Check if we have any saved configs to determine tab labels
	configs, _ := m.configManager.ListConfigs()
	configTabLabel := "Previous Configurations"
	if len(configs) > 0 {
		configTabLabel += " (" + string(rune(len(configs)+48)) + ")" // Convert to string number
	}

	// Render tabs
	for i, tab := range []string{configTabLabel, "New Configuration"} {
		var style lipgloss.Style
		isFirst, isActive := i == 0, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "┴"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	// Join tabs horizontally
	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Add gap filler to the right of tabs
	gap := m.tabGap()
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	// Content area
	content := m.renderActiveTabContent()

	// Combine tabs and content
	doc := lipgloss.JoinVertical(lipgloss.Left, row, windowStyle.Render(content))

	// Add help text at bottom
	helpText := m.renderHelp()

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, doc, helpText))
}

func (m MainModel) tabGap() string {
	tabsWidth := len("Previous Configurations") + len("New Configuration") + 4 // approx tab width
	availableWidth := m.windowWidth - tabsWidth
	if availableWidth < 0 {
		availableWidth = 0
	}
	return inactiveTabStyle.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Render(lipgloss.NewStyle().Width(availableWidth).Render(""))
}

func (m MainModel) renderActiveTabContent() string {
	switch m.activeTab {
	case configsTab:
		return m.configListModel.View()
	case newConfigTab:
		return m.newConfigModel.View()
	default:
		return "Unknown tab"
	}
}

func (m MainModel) renderHelp() string {
	helpItems := []string{}

	switch m.activeTab {
	case configsTab:
		helpItems = append(helpItems,
			"[←/→] Switch tabs",
			"[↑/↓] Navigate",
			"[enter] Launch",
			"[e] Edit",
			"[d] Delete",
			"[q] Quit",
		)
	case newConfigTab:
		helpItems = append(helpItems,
			"[←/→] Switch tabs",
			"[tab] Switch panes",
			"[space] Toggle selection",
			"[enter] Confirm",
			"[q] Quit",
		)
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, helpItems...))
}

// GetResult returns the selected configuration or extensions after the UI finishes
func (m MainModel) GetResult() (*config.SavedConfig, []string, bool) {
	if m.cancelled {
		return nil, nil, false
	}
	return m.selectedConfig, m.selectedExtensions, m.finished
}

// RunMainInterface is the main entry point for the configuration interface
func RunMainInterface(extManager *extensions.Manager) (*config.SavedConfig, []string, error) {
	// First, show the menu to select or create a configuration
	menuModel := NewMenuModel(extManager)
	program := tea.NewProgram(menuModel, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return nil, nil, err
	}

	menuResult, ok := finalModel.(*MenuModel)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected model type")
	}

	selectedConfig, createNew, success := menuResult.GetResult()
	if !success {
		return nil, nil, nil // User cancelled
	}

	if selectedConfig != nil {
		// User selected an existing configuration
		return selectedConfig, nil, nil
	}

	if createNew {
		// User wants to create a new configuration - show extension selector
		selectorModel := NewExtensionSelectorModel(extManager, nil)
		program = tea.NewProgram(selectorModel, tea.WithAltScreen())

		finalModel, err := program.Run()
		if err != nil {
			return nil, nil, err
		}

		selectorResult, ok := finalModel.(*ExtensionSelectorModel)
		if !ok {
			return nil, nil, fmt.Errorf("unexpected selector model type")
		}

		extensions, configName, description, success := selectorResult.GetResult()
		if !success || len(extensions) == 0 {
			return nil, nil, nil // User cancelled or no extensions selected
		}

		// Save the configuration if named
		if configName != "" {
			currentConfig := config.NewConfig()
			savedConfig := config.CreateConfigFromCurrent(currentConfig, extensions, configName, description)
			_ = config.NewConfigManager().SaveConfig(savedConfig)
		}

		return nil, extensions, nil
	}

	return nil, nil, nil
}

// Keep the old function for backward compatibility but mark as deprecated
// RunMainInterface is the main entry point for the tabbed interface
func RunTabInterface(extManager *extensions.Manager) (*config.SavedConfig, []string, error) {
	model := NewMainModel(extManager)
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return nil, nil, err
	}

	if mainModel, ok := finalModel.(MainModel); ok {
		selectedConfig, extensions, success := mainModel.GetResult()
		if !success {
			return nil, nil, nil // User cancelled
		}
		return selectedConfig, extensions, nil
	}

	return nil, nil, nil
}
