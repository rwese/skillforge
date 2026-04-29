package agents

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectableAgent represents an agent that can be toggled in the selector.
type SelectableAgent struct {
	Name          string
	DefaultGlobal string
	DefaultLocal  string
	Detected      bool  // True if skill directory exists
	Selected      bool  // Current selection state
	Configured    bool  // True if already in config
}

// Bubbletea model
type model struct {
	agents   []SelectableAgent
	selected []bool
	cursor   int
	quitting bool
}

func initialModel(agents []SelectableAgent) model {
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}
	return model{
		agents:   agents,
		selected: selected,
		cursor:   0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.agents) - 1
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.agents) {
				m.cursor = 0
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			for i := range m.selected {
				if m.agents[i].Configured {
					m.selected[i] = !m.selected[i]
				}
			}
		case "s":
			for i := range m.selected {
				m.selected[i] = true
			}
		case "n":
			for i := range m.selected {
				m.selected[i] = false
			}
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var s string

	s += "\n"
	s += headerStyle.Render("  Select agents to configure") + "\n"
	s += headerStyle.Render("  ─────────────────────────") + "\n"
	s += "\n"
	s += helpStyle.Render("  ↑↓ Navigate   Space Toggle   Enter Confirm   Q Quit") + "\n"
	s += helpStyle.Render("  A Toggle all   S Select all   N Deselect all") + "\n"
	s += "\n"

	for i, agent := range m.agents {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render(" ▸")
		}

		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = selectedStyle.Render("[✓]")
		}

		name := agent.Name
		nameStyle := lipgloss.NewStyle()
		if m.selected[i] {
			nameStyle = selectedStyle
		}

		path := ContractPath(ExpandPath(agent.DefaultGlobal))

		// Determine badge
		badge := ""
		if agent.Configured {
			badge = " " + configuredStyle.Render("●")
		} else if agent.Detected {
			badge = " " + detectedStyle.Render("○")
		}

		// Use lipgloss for proper width calculations
		namePadded := nameStyle.Width(10).Render(name)
		pathPadded := lipgloss.NewStyle().Width(30).Render(path)

		s += fmt.Sprintf("%s %s %s %s%s\n", cursor, checkbox, namePadded, pathPadded, badge)
	}

	s += "\n"

	return s
}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71"))

	configuredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71"))

	detectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB"))
)

// SelectAgents displays an interactive checkbox selector for agents.
// Returns the list of agents the user selected.
func SelectAgents(agents []SelectableAgent) []string {
	if len(agents) == 0 {
		return nil
	}

	m := initialModel(agents)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil
	}

	result := finalModel.(model)

	if result.quitting {
		return nil
	}

	var selected []string
	for i, s := range result.selected {
		if s {
			selected = append(selected, result.agents[i].Name)
		}
	}

	return selected
}
