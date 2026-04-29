package agents

import (
	"fmt"
	"os"
	"strings"

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

// Checkbox styles
var (
	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECF0F1"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71")).
			Bold(true)

	deselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D"))

	selectedIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#2ECC71"))

	deselectedIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7F8C8D"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D"))

	detectedBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB"))

	configuredBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71"))

	dimBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D"))
)

// SelectAgents displays an interactive checkbox selector for agents.
// Returns the list of agents the user selected.
func SelectAgents(agents []SelectableAgent) []string {
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}

	cursor := 0

	for {
		clearScreen()
		renderSelector(agents, selected, cursor)

		key := readKey()
		switch key {
		case "up", "k":
			cursor = (cursor - 1 + len(agents)) % len(agents)
		case "down", "j":
			cursor = (cursor + 1) % len(agents)
		case " ":
			selected[cursor] = !selected[cursor]
		case "a":
			// Toggle all
			for i := range selected {
				if agents[i].Configured {
					selected[i] = !selected[i]
				}
			}
		case "s":
			// Select all configured
			for i := range selected {
				selected[i] = true
			}
		case "n":
			// Deselect all
			for i := range selected {
				selected[i] = false
			}
		case "enter":
			// Collect selected agent names
			var result []string
			for i, s := range selected {
				if s {
					result = append(result, agents[i].Name)
				}
			}
			return result
		case "q", "ctrl+c":
			return nil
		}
	}
}

func renderSelector(agents []SelectableAgent, selected []bool, cursor int) {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  Use arrow keys to navigate, Space to toggle, Enter to confirm, Q to quit\n"))
	sb.WriteString(helpStyle.Render("  [a] select/deselect all configured  [s] select all  [n] deselect all\n"))
	sb.WriteString("\n")

	for i, agent := range agents {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		// Checkbox
		var checkbox string
		if selected[i] {
			checkbox = selectedIndicator.Render("[x]")
		} else {
			checkbox = deselectedIndicator.Render("[ ]")
		}

		// Agent name
		var name string
		if selected[i] {
			name = selectedStyle.Render(agent.Name)
		} else {
			name = checkboxStyle.Render(agent.Name)
		}

		// Path info
		globalPath := ContractPath(ExpandPath(agent.DefaultGlobal))
		pathInfo := dimBadge.Render(globalPath)

		// Badges
		var badges []string
		if agent.Detected && !agent.Configured {
			badges = append(badges, detectedBadge.Render("(detected)"))
		}
		if agent.Configured {
			badges = append(badges, configuredBadge.Render("(configured)"))
		}

		badgeStr := ""
		if len(badges) > 0 {
			badgeStr = " " + strings.Join(badges, " ")
		}

		sb.WriteString(fmt.Sprintf("%s%s %s %s%s\n", prefix, checkbox, name, pathInfo, badgeStr))
	}

	fmt.Print(sb.String())
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func readKey() string {
	// Read first byte
	var buf [1]byte
	os.Stdin.Read(buf[:])
	
	first := buf[0]
	
	// Handle escape sequences (arrow keys)
	if first == 27 {
		// Read next two bytes for arrow keys
		os.Stdin.Read(buf[:])
		os.Stdin.Read(buf[:])
		if buf[0] == 65 {
			return "up"
		}
		if buf[0] == 66 {
			return "down"
		}
		if buf[0] == 67 {
			return "right"
		}
		if buf[0] == 68 {
			return "left"
		}
		return ""
	}
	
	// Handle Enter
	if first == 13 {
		return "enter"
	}
	
	// Handle Ctrl+C
	if first == 3 {
		return "ctrl+c"
	}
	
	return string([]byte{first})
}
