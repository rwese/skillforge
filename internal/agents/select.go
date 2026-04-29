package agents

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
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
	// Set terminal to raw mode for byte-by-byte reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback: try without raw mode
		return selectAgentsFallback(agents)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	return selectAgentsLoop(agents)
}

func selectAgentsFallback(agents []SelectableAgent) []string {
	// Fallback without raw mode - uses buffered input
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}

	cursor := 0

	for {
		clearScreen()
		renderSelector(agents, selected, cursor)
		fmt.Print("\nSelect agents and press Enter, or q to quit: ")

		var input string
		fmt.Scanln(&input)

		switch input {
		case "q", "Q":
			return nil
		case "":
			var result []string
			for i, s := range selected {
				if s {
					result = append(result, agents[i].Name)
				}
			}
			return result
		case "n", "N":
			for i := range selected {
				selected[i] = false
			}
		case "s", "S":
			for i := range selected {
				selected[i] = true
			}
		}
	}
}

func selectAgentsLoop(agents []SelectableAgent) []string {
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
	var buf [1]byte
	os.Stdin.Read(buf[:])

	first := buf[0]

	// Handle escape sequences (arrow keys)
	if first == 27 {
		os.Stdin.Read(buf[:])
		if buf[0] == 91 {
			os.Stdin.Read(buf[:])
			switch buf[0] {
			case 65:
				return "up"
			case 66:
				return "down"
			case 67:
				return "right"
			case 68:
				return "left"
			}
		}
		return ""
	}

	// Handle Enter
	if first == 13 || first == 10 {
		return "enter"
	}

	// Handle Ctrl+C
	if first == 3 {
		return "ctrl+c"
	}

	return string([]byte{first})
}
