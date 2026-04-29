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

	cursorPrefix = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71"))

	itemLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECF0F1"))
)

// SelectAgents displays an interactive checkbox selector for agents.
// Returns the list of agents the user selected.
func SelectAgents(agents []SelectableAgent) []string {
	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24 // Default fallback
	}

	// Set terminal to raw mode for byte-by-byte reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback: try without raw mode
		return selectAgentsFallback(agents)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	return selectAgentsLoop(agents, width, height)
}

func selectAgentsFallback(agents []SelectableAgent) []string {
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}

	cursor := 0

	for {
		clearScreen()
		renderSelector(agents, selected, cursor, 80, 24)
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

func selectAgentsLoop(agents []SelectableAgent, width, height int) []string {
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}

	cursor := 0

	for {
		clearScreen()
		renderSelector(agents, selected, cursor, width, height)

		key := readKey()
		switch key {
		case "up", "k":
			cursor = (cursor - 1 + len(agents)) % len(agents)
		case "down", "j":
			cursor = (cursor + 1) % len(agents)
		case " ":
			selected[cursor] = !selected[cursor]
		case "a":
			for i := range selected {
				if agents[i].Configured {
					selected[i] = !selected[i]
				}
			}
		case "s":
			for i := range selected {
				selected[i] = true
			}
		case "n":
			for i := range selected {
				selected[i] = false
			}
		case "enter":
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

func renderSelector(agents []SelectableAgent, selected []bool, cursor int, width, height int) {
	// Use available width for content
	contentWidth := width - 4 // padding

	var sb strings.Builder

	// Title bar
	sb.WriteString(helpStyle.Render("\n  ── Select agents to configure ──\n\n"))

	// Help text
	sb.WriteString(helpStyle.Render("  [↑/↓] Navigate  [Space] Toggle  [Enter] Confirm  [Q] Quit\n"))
	sb.WriteString(helpStyle.Render("  [A] Toggle all  [S] Select all  [N] Deselect all\n"))
	sb.WriteString("\n")

	// Render each agent
	for i, agent := range agents {
		// Build the line piece by piece
		var line strings.Builder

		// Cursor indicator
		if i == cursor {
			line.WriteString(cursorPrefix.Render("  ▸ "))
		} else {
			line.WriteString("    ")
		}

		// Checkbox
		if selected[i] {
			line.WriteString(selectedIndicator.Render("[✓]"))
		} else {
			line.WriteString(deselectedIndicator.Render("[ ]"))
		}
		line.WriteString("  ")

		// Agent name
		if selected[i] {
			line.WriteString(selectedStyle.Render(agent.Name))
		} else {
			line.WriteString(checkboxStyle.Render(agent.Name))
		}

		// Path (truncated if needed)
		globalPath := ContractPath(ExpandPath(agent.DefaultGlobal))
		pathLen := lipgloss.Width(globalPath)
		availableForPath := contentWidth - lipgloss.Width(line.String()) - 4

		if pathLen > availableForPath {
			// Truncate path
			maxPathLen := availableForPath - 3
			if maxPathLen > 0 {
				globalPath = globalPath[:maxPathLen] + "…"
			}
		}
		line.WriteString("  ")
		line.WriteString(dimBadge.Render(globalPath))

		// Badges
		if agent.Configured {
			line.WriteString(" ")
			line.WriteString(configuredBadge.Render("●"))
		} else if agent.Detected {
			line.WriteString(" ")
			line.WriteString(detectedBadge.Render("○"))
		}

		sb.WriteString(line.String())
		sb.WriteString("\n")
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
