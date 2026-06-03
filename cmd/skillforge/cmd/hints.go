package cmd

import (
	"fmt"
	"os"
	"strings"
)

// execName is the executable name, updated after cobra parses args.
// Defaults to "skillforge" but will be set to the actual command name.
var execName = "skillforge"

// SetExecName sets the executable name (called from root.go init).
func SetExecName(name string) {
	execName = name
}

// cmdName returns the command name for hints, handling potential path prefixes.
func cmdName() string {
	return execName
}

// Hints provides contextual help for error messages.
type Hint struct {
	Title string
	Lines []string
}

// FormatHint formats a hint for display.
// Output:
//
//	Hint:
//	  • Run: $0 target add <name> <globalPath> <localPath> -e
//	  • Example: $0 target add pi ~/.pi/agent/skills .pi/skills -e
func FormatHint(hint Hint) string {
	if len(hint.Lines) == 0 {
		return ""
	}

	var result string
	if hint.Title != "" {
		result = fmt.Sprintf("\nHint (%s):\n", hint.Title)
	} else {
		result = "\nHint:\n"
	}

	for _, line := range hint.Lines {
		// Replace $0 with actual command name
		line = strings.ReplaceAll(line, "$0", cmdName())
		result += fmt.Sprintf("  • %s\n", line)
	}

	return result
}

// PrintHint prints a formatted hint to stderr.
func PrintHint(hint Hint) {
	fmt.Fprint(os.Stderr, FormatHint(hint))
}

// Common hints used across commands.
var (
	HintNoTargets = Hint{
		Title: "no targets configured",
		Lines: []string{
			"Run: $0 target add <name> <globalPath> <localPath> -e",
			"Example: $0 target add pi ~/.pi/agent/skills .pi/skills -e",
		},
	}

	HintNoRepos = Hint{
		Title: "no repositories configured",
		Lines: []string{
			"Run: $0 repo add <name> <url>",
			"Example: $0 repo add grimoire https://github.com/rwese/agents-grimoire.git",
		},
	}

	HintRepoNotCached = Hint{
		Title: "repository not cached",
		Lines: []string{
			"Run: $0 repo update",
			"Example: $0 repo update grimoire",
		},
	}

	HintSkillNotFound = Hint{
		Title: "skill not found",
		Lines: []string{
			"Run: $0 repo update to refresh skills",
			"Run: $0 skill search <query> to search",
		},
	}

	HintSearchNoResults = Hint{
		Title: "no search results",
		Lines: []string{
			"Run: $0 repo update to refresh skills",
			"Try a different search term",
		},
	}

	HintTargetExists = Hint{
		Title: "target already exists",
		Lines: []string{
			"Use a different name: $0 target add <newname> <globalPath> <localPath> -e",
			"View targets: $0 target list",
		},
	}

	HintTargetNotFound = Hint{
		Title: "target not found",
		Lines: []string{
			"View targets: $0 target list",
			"Add new target: $0 target add <name> <globalPath> <localPath> -e",
		},
	}

	HintRepoExists = Hint{
		Title: "repository already exists",
		Lines: []string{
			"Run: $0 repo update to refresh",
			"View repos: $0 repo list",
		},
	}

	HintRepoNotFound = Hint{
		Title: "repository not found",
		Lines: []string{
			"View repos: $0 repo list",
			"Add new repo: $0 repo add <name> <url>",
		},
	}

	HintSkillNotInstalled = Hint{
		Title: "skill not installed",
		Lines: []string{
			"View installed: $0 skill list",
			"Install skill: $0 skill install <name>",
		},
	}
)
