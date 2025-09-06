package ui

import (
	"fmt"

	"github.com/ahacop/pgbox/pkg/extensions"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle      = lipgloss.NewStyle().MarginLeft(2)
	paginationStyle = lipgloss.NewStyle().PaddingLeft(4)
	helpStyle       = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("241"))
)

type item struct {
	title, desc string
	name        string
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

type listModel struct {
	list     list.Model
	choice   string // Keep for backward compatibility
	quitting bool
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.name
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	if m.choice != "" {
		return fmt.Sprintf("Selected: %s\n", m.choice)
	}
	if m.quitting {
		return "Bye!\n"
	}
	return "\n" + m.list.View()
}

func RunExtensionSelector(exts []extensions.Extension) ([]string, error) {
	items := make([]list.Item, len(exts))
	for i, ext := range exts {
		items[i] = item{
			title: ext.Name,
			desc:  fmt.Sprintf("%s (%s)", ext.Description, ext.Kind),
			name:  ext.Name,
		}
	}

	const defaultWidth = 20

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, 14)
	l.Title = "Choose an extension"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := listModel{list: l}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := finalModel.(listModel); ok {
		if m.choice == "" {
			return nil, fmt.Errorf("no extension selected")
		}
		return []string{m.choice}, nil
	}

	return nil, fmt.Errorf("unexpected error")
}
