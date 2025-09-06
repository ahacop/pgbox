package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ahacop/pgbox/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigListModel handles the saved configurations list view
type ConfigListModel struct {
	configManager *config.ConfigManager
	configs       []*config.SavedConfig
	cursor        int

	// State
	ShowConfirmDelete bool // Export for main model access
	configToDelete    *config.SavedConfig

	// Result
	selectedConfig *config.SavedConfig
	editConfig     *config.SavedConfig
	finished       bool
	cancelled      bool
}

// NewConfigListModel creates a new configuration list model
func NewConfigListModel(configManager *config.ConfigManager) *ConfigListModel {
	model := &ConfigListModel{
		configManager: configManager,
		cursor:        0,
	}
	model.loadConfigs()
	return model
}

func (m *ConfigListModel) loadConfigs() {
	configs, err := m.configManager.ListConfigs()
	if err != nil {
		m.configs = []*config.SavedConfig{}
	} else {
		m.configs = configs
	}
}

func (m *ConfigListModel) Init() tea.Cmd {
	return nil
}

func (m *ConfigListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle delete confirmation first
		if m.ShowConfirmDelete {
			switch msg.String() {
			case "y", "Y":
				// Delete the configuration
				if m.configToDelete != nil {
					err := m.configManager.DeleteConfig(m.configToDelete.Name)
					if err == nil {
						m.loadConfigs() // Reload configurations
						if m.cursor >= len(m.configs) && len(m.configs) > 0 {
							m.cursor = len(m.configs) - 1
						}
					}
				}
				m.ShowConfirmDelete = false
				m.configToDelete = nil
				return m, nil

			case "n", "N", "esc":
				m.ShowConfirmDelete = false
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
			if m.cursor < len(m.configs)-1 {
				m.cursor++
			}

		case "enter":
			// Launch selected configuration
			if len(m.configs) > 0 && m.cursor < len(m.configs) {
				selected := m.configs[m.cursor]
				// Update last used timestamp
				_ = m.configManager.UpdateLastUsed(selected.Name)
				m.selectedConfig = selected
				m.finished = true
				return m, tea.Quit
			}

		case "e":
			// Edit selected configuration
			if len(m.configs) > 0 && m.cursor < len(m.configs) {
				m.editConfig = m.configs[m.cursor]
				m.finished = true
				return m, tea.Quit
			}

		case "d":
			// Delete selected configuration (with confirmation)
			if len(m.configs) > 0 && m.cursor < len(m.configs) {
				m.configToDelete = m.configs[m.cursor]
				m.ShowConfirmDelete = true
			}

		case "r":
			// Refresh configuration list
			m.loadConfigs()
		}
	}

	return m, nil
}

func (m *ConfigListModel) View() string {
	if m.ShowConfirmDelete && m.configToDelete != nil {
		return m.renderDeleteConfirmation()
	}

	if len(m.configs) == 0 {
		return m.renderEmptyState()
	}

	return m.renderConfigList()
}

func (m *ConfigListModel) renderEmptyState() string {
	return lipgloss.NewStyle().
		Padding(4, 2).
		Align(lipgloss.Center).
		Render(
			"📁 No saved configurations found\n\n" +
				"Press → or Tab to create your first configuration!\n\n" +
				"Once you create and successfully launch a configuration,\n" +
				"it will be saved here for quick access later.",
		)
}

func (m *ConfigListModel) renderConfigList() string {
	var content strings.Builder

	// Header
	content.WriteString("📁 Saved Configurations")
	if len(m.configs) > 0 {
		content.WriteString(fmt.Sprintf(" (%d)", len(m.configs)))
	}
	content.WriteString("\n\n")

	// List configurations
	for i, cfg := range m.configs {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		// Configuration name and extension count
		nameStyle := lipgloss.NewStyle()
		if m.cursor == i {
			nameStyle = nameStyle.Foreground(lipgloss.Color("12")).Bold(true)
		}

		name := nameStyle.Render(cfg.Name)
		extensionCount := fmt.Sprintf("(%d extensions)", len(cfg.Extensions))

		content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, name, extensionCount))

		// Details (show more info for selected item)
		if m.cursor == i {
			details := m.renderConfigDetails(cfg)
			// Indent details
			for _, line := range strings.Split(details, "\n") {
				if line != "" {
					content.WriteString("    " + line + "\n")
				}
			}
		} else {
			// Show minimal details for non-selected items
			lastUsed := formatTimeAgo(cfg.LastUsed)
			content.WriteString(fmt.Sprintf("    Port: %s • Last used: %s\n", cfg.Port, lastUsed))
		}

		content.WriteString("\n")
	}

	return content.String()
}

func (m *ConfigListModel) renderConfigDetails(cfg *config.SavedConfig) string {
	var details strings.Builder

	// Basic info
	details.WriteString(fmt.Sprintf("Port: %s • PostgreSQL: %s", cfg.Port, cfg.PgMajor))
	if cfg.Database != "" {
		details.WriteString(fmt.Sprintf(" • DB: %s", cfg.Database))
	}
	details.WriteString("\n")

	// Description if available
	if cfg.Description != "" {
		details.WriteString(fmt.Sprintf("Description: %s\n", cfg.Description))
	}

	// Extensions (show first few, then "...")
	if len(cfg.Extensions) > 0 {
		extList := strings.Join(cfg.Extensions[:min(3, len(cfg.Extensions))], ", ")
		if len(cfg.Extensions) > 3 {
			extList += fmt.Sprintf(" and %d more", len(cfg.Extensions)-3)
		}
		details.WriteString(fmt.Sprintf("Extensions: %s\n", extList))
	}

	// Timestamps
	created := cfg.CreatedAt.Format("2006-01-02 15:04")
	lastUsed := formatTimeAgo(cfg.LastUsed)
	details.WriteString(fmt.Sprintf("Created: %s • Last used: %s", created, lastUsed))

	return details.String()
}

func (m *ConfigListModel) renderDeleteConfirmation() string {
	if m.configToDelete == nil {
		return "Error: no configuration to delete"
	}

	confirmStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border
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

	return confirmStyle.Render(content)
}

// GetResult returns the result of the configuration list interaction
func (m *ConfigListModel) GetResult() (*config.SavedConfig, *config.SavedConfig, bool) {
	if m.cancelled {
		return nil, nil, true // cancelled, so return empty results but finished=true
	}
	return m.selectedConfig, m.editConfig, m.finished
}

// Helper functions
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	case duration < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
