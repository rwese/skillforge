package agents

import (
	"fmt"
	"os"

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

// SelectAgents displays an interactive checkbox selector for agents.
// Returns the list of agents the user selected.
func SelectAgents(agents []SelectableAgent) []string {
	// Set terminal to raw mode for byte-by-byte reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return selectAgentsFallback(agents)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	return selectAgentsLoop(agents)
}

func selectAgentsFallback(agents []SelectableAgent) []string {
	selected := make([]bool, len(agents))
	for i := range agents {
		selected[i] = agents[i].Selected
	}

	for {
		clearScreen()
		renderSelector(agents, selected, 0)
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

func renderSelector(agents []SelectableAgent, selected []bool, cursor int) {
	fmt.Println()
	fmt.Println("  Select agents to configure")
	fmt.Println("  ─────────────────────────")

	fmt.Println()
	fmt.Println("  [↑/↓] Navigate   [Space] Toggle   [Enter] Confirm   [Q] Quit")
	fmt.Println("  [A] Toggle all  [S] Select all  [N] Deselect all")
	fmt.Println()

	// Simple columnar output - each agent on its own line
	for i, agent := range agents {
		// Cursor indicator
		prefix := "    "
		if i == cursor {
			prefix = "  > "
		}

		// Checkbox
		checkbox := "[ ]"
		if selected[i] {
			checkbox = "[x]"
		}

		// Path (shortened)
		path := ContractPath(ExpandPath(agent.DefaultGlobal))
		if len(path) > 30 {
			path = path[:27] + "..."
		}

		// Badge
		badge := ""
		if agent.Configured {
			badge = " [configured]"
		} else if agent.Detected {
			badge = " [detected]"
		}

		// Simple row without padding
		fmt.Printf("%s%s %s %s%s\n", prefix, checkbox, agent.Name, path, badge)
	}
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
