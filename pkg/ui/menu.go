package ui

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/pkg/extensions"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuModel represents the main menu interface
type MenuModel struct {
	configManager *config.ConfigManager
	extManager    *extensions.Manager

	// Menu items
	configs []*config.SavedConfig
	cursor  int

	// State
	selectedConfig    *config.SavedConfig
	createNew         bool
	showDeleteConfirm bool
	configToDelete    *config.SavedConfig
	finished          bool
	cancelled         bool

	// Window sizing
	width  int
	height int
}

// NewMenuModel creates a new menu model
func NewMenuModel(extManager *extensions.Manager) *MenuModel {
	model := &MenuModel{
		configManager: config.NewConfigManager(),
		extManager:    extManager,
		width:         80,
		height:        24,
	}
	model.loadConfigs()
	return model
}

func (m *MenuModel) loadConfigs() {
	configs, err := m.configManager.ListConfigs()
	if err != nil {
		m.configs = []*config.SavedConfig{}
	} else {
		m.configs = configs
	}

	// Reset cursor if needed
	if m.cursor >= len(m.configs)+1 { // +1 for "Create New" option
		m.cursor = 0
	}
}

func (m *MenuModel) Init() tea.Cmd {
	return nil
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Handle delete confirmation
		if m.showDeleteConfirm {
			switch msg.String() {
			case "y", "Y":
				if m.configToDelete != nil {
					_ = m.configManager.DeleteConfig(m.configToDelete.Name)
					m.loadConfigs()
				}
				m.showDeleteConfirm = false
				m.configToDelete = nil
				return m, nil

			case "n", "N", "esc":
				m.showDeleteConfirm = false
				m.configToDelete = nil
				return m, nil
			}
			return m, nil
		}

		// Regular key handling
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			m.finished = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxItems := len(m.configs) + 1 // +1 for "Create New"
			if m.cursor < maxItems-1 {
				m.cursor++
			}

		case "enter":
			if m.cursor == len(m.configs) {
				// "Create New" option selected
				m.createNew = true
				m.finished = true
				return m, tea.Quit
			} else if m.cursor < len(m.configs) {
				// Existing config selected
				selected := m.configs[m.cursor]
				_ = m.configManager.UpdateLastUsed(selected.Name)
				m.selectedConfig = selected
				m.finished = true
				return m, tea.Quit
			}

		case "d":
			// Delete selected configuration (only for existing configs)
			if m.cursor < len(m.configs) {
				m.configToDelete = m.configs[m.cursor]
				m.showDeleteConfirm = true
			}

		case "r":
			// Refresh
			m.loadConfigs()
		}
	}

	return m, nil
}

func (m *MenuModel) View() string {
	if m.showDeleteConfirm {
		return m.renderDeleteConfirmation()
	}

	return m.renderMenu()
}

func (m *MenuModel) renderMenu() string {
	var content strings.Builder

	// Header
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render("🗂️  PostgreSQL Configuration Manager"))
	content.WriteString("\n\n")

	if len(m.configs) == 0 {
		// No saved configurations
		content.WriteString("No saved configurations found.\n\n")
	} else {
		// List saved configurations
		content.WriteString(fmt.Sprintf("Saved configurations (%d):\n\n", len(m.configs)))

		for i, cfg := range m.configs {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}

			nameStyle := lipgloss.NewStyle()
			if m.cursor == i {
				nameStyle = nameStyle.Foreground(lipgloss.Color("12")).Bold(true)
			}

			name := nameStyle.Render(cfg.Name)
			extensionCount := fmt.Sprintf("(%d extensions)", len(cfg.Extensions))

			content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, name, extensionCount))

			if m.cursor == i {
				// Show details for selected item
				details := fmt.Sprintf("    Port: %s • PostgreSQL: %s • Last used: %s",
					cfg.Port, cfg.PgMajor, formatTimeAgo(cfg.LastUsed))
				content.WriteString(details + "\n")

				if cfg.Description != "" {
					content.WriteString(fmt.Sprintf("    %s\n", cfg.Description))
				}

				// Show some extensions
				if len(cfg.Extensions) > 0 {
					extList := strings.Join(cfg.Extensions[:min(4, len(cfg.Extensions))], ", ")
					if len(cfg.Extensions) > 4 {
						extList += "..."
					}
					content.WriteString(fmt.Sprintf("    Extensions: %s\n", extList))
				}
			}
			content.WriteString("\n")
		}
	}

	// "Create New" option
	cursor := "  "
	if m.cursor == len(m.configs) {
		cursor = "> "
	}

	createNewStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	if m.cursor == len(m.configs) {
		createNewStyle = createNewStyle.Bold(true)
	}

	createNewText := createNewStyle.Render("🆕 Create New Configuration")
	content.WriteString(fmt.Sprintf("%s%s\n", cursor, createNewText))

	if m.cursor == len(m.configs) {
		content.WriteString("    Start with a fresh configuration and select extensions\n")
	}

	// Help text
	content.WriteString("\n")
	content.WriteString(m.renderHelp())

	// Center and pad content
	return lipgloss.NewStyle().
		Padding(2, 4).
		Render(content.String())
}

func (m *MenuModel) renderDeleteConfirmation() string {
	if m.configToDelete == nil {
		return "Error: no configuration to delete"
	}

	confirmStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 4).
		Align(lipgloss.Center)

	content := fmt.Sprintf(
		"Delete Configuration?\n\n"+
			"This will permanently delete:\n"+
			"  %s\n\n"+
			"This action cannot be undone.\n\n"+
			"[y] Yes, delete    [n] Cancel",
		m.configToDelete.Name,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, confirmStyle.Render(content))
}

func (m *MenuModel) renderHelp() string {
	helpItems := []string{
		"[↑/↓] Navigate",
		"[Enter] Select",
	}

	if len(m.configs) > 0 && m.cursor < len(m.configs) {
		helpItems = append(helpItems, "[d] Delete")
	}

	helpItems = append(helpItems, "[r] Refresh", "[q] Quit")

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(strings.Join(helpItems, " • "))
}

// GetResult returns the result of the menu interaction
func (m *MenuModel) GetResult() (*config.SavedConfig, bool, bool) {
	if m.cancelled {
		return nil, false, true // cancelled, so createNew=false but finished=true
	}
	return m.selectedConfig, m.createNew, m.finished
}
