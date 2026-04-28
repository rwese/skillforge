package main

import (
	"fmt"
	"os"
)

// Hints provides contextual help for error messages.
type Hint struct {
	Title   string
	Lines   []string
}

// FormatHint formats a hint for display.
// Output:
//   Hint:
//     • Run: skillforge target add <name> <path> -e
//     • Example: skillforge target add pi ~/.pi/agent/skills/ -e
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
			"Run: skillforge target add <name> <path> -e",
			"Example: skillforge target add pi ~/.pi/agent/skills/ -e",
		},
	}

	HintNoRepos = Hint{
		Title: "no repositories configured",
		Lines: []string{
			"Run: skillforge repo add <name> <url>",
			"Example: skillforge repo add grimoire https://github.com/rwese/agents-grimoire.git",
		},
	}

	HintRepoNotCached = Hint{
		Title: "repository not cached",
		Lines: []string{
			"Run: skillforge repo update",
			"Example: skillforge repo update grimoire",
		},
	}

	HintSkillNotFound = Hint{
		Title: "skill not found",
		Lines: []string{
			"Run: skillforge repo update to refresh skills",
			"Run: skillforge skill search <query> to search",
		},
	}

	HintSearchNoResults = Hint{
		Title: "no search results",
		Lines: []string{
			"Run: skillforge repo update to refresh skills",
			"Try a different search term",
		},
	}

	HintTargetExists = Hint{
		Title: "target already exists",
		Lines: []string{
			"Use a different name: skillforge target add <newname> <path> -e",
			"View targets: skillforge target list",
		},
	}

	HintTargetNotFound = Hint{
		Title: "target not found",
		Lines: []string{
			"View targets: skillforge target list",
			"Add new target: skillforge target add <name> <path> -e",
		},
	}

	HintRepoExists = Hint{
		Title: "repository already exists",
		Lines: []string{
			"Run: skillforge repo update to refresh",
			"View repos: skillforge repo list",
		},
	}

	HintRepoNotFound = Hint{
		Title: "repository not found",
		Lines: []string{
			"View repos: skillforge repo list",
			"Add new repo: skillforge repo add <name> <url>",
		},
	}

	HintSkillNotInstalled = Hint{
		Title: "skill not installed",
		Lines: []string{
			"View installed: skillforge skill list",
			"Install skill: skillforge skill install <name>",
		},
	}
)
