package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// OutputFormat represents the output format type.
type OutputFormat int

const (
	formatText OutputFormat = iota
	formatTable
	formatCompact
	formatJSON
)

// parseFormat converts a format string to OutputFormat.
func parseFormat(format string) OutputFormat {
	switch strings.ToLower(format) {
	case "table":
		return formatTable
	case "compact":
		return formatCompact
	case "json":
		return formatJSON
	default:
		// Auto-detect: use compact if not a terminal
		if !isPiped() {
			return formatTable
		}
		return formatCompact
	}
}

// isPiped checks if stdout is being piped/redirected.
func isPiped() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return true // Assume piped if we can't check
	}
	mode := fileInfo.Mode()
	return mode&os.ModeCharDevice == 0
}

// --- Table Styling ---

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#34495E"))

var headerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7F8C8D")).
	Bold(true).
	Align(lipgloss.Left)

var cellStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ECF0F1")).
	Align(lipgloss.Left)

// --- Target Output Types ---

// TargetOutput represents a target for JSON output.
type TargetOutput struct {
	Name       string `json:"name"`
	GlobalPath string `json:"globalPath"`
	LocalPath  string `json:"localPath"`
	Enabled    bool   `json:"enabled"`
}

// --- Skill Output Types ---

// SkillOutput represents a skill for JSON output.
type SkillOutput struct {
	Name        string `json:"name"`
	Commit      string `json:"commit"`
	Target      string `json:"target"`
	Repo        string `json:"repo"`
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
}

// --- Repo Output Types ---

// RepoOutput represents a repository for JSON output.
type RepoOutput struct {
	Name       string `json:"name"`
	Alias      string `json:"alias,omitempty"`
	URL        string `json:"url"`
	Branch     string `json:"branch"`
	SkillCount int    `json:"skill_count"`
	Updated    string `json:"updated,omitempty"`
}

// --- Search Result Output ---

// SearchResultOutput represents search results for JSON output.
type SearchResultOutput struct {
	Query   string        `json:"query"`
	Results []SkillOutput `json:"results"`
	Count   int           `json:"count"`
}

// printJSON prints data as formatted JSON.
func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// --- Target List Formatting ---

// formatTargetTable formats targets as a table.
func formatTargetTable(targets []TargetOutput) string {
	if len(targets) == 0 {
		return "No targets configured."
	}

	// Sort targets by name
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})

	var b strings.Builder

	// Calculate column widths
	nameWidth := 10
	globalPathWidth := 20
	localPathWidth := 20
	for _, t := range targets {
		if len(t.Name) > nameWidth {
			nameWidth = len(t.Name)
		}
		if len(t.GlobalPath) > globalPathWidth {
			globalPathWidth = len(t.GlobalPath)
		}
		if len(t.LocalPath) > localPathWidth {
			localPathWidth = len(t.LocalPath)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
		nameWidth, "TARGET",
		globalPathWidth, "GLOBAL",
		localPathWidth, "LOCAL",
		"STATUS")
	b.WriteString(header)
	b.WriteString("\n")

	// Print separator
	sep := strings.Repeat("─", nameWidth) + "  " +
		strings.Repeat("─", globalPathWidth) + "  " +
		strings.Repeat("─", localPathWidth) + "  " +
		"───────"
	b.WriteString(sep)
	b.WriteString("\n")

	// Print rows
	for _, t := range targets {
		status := "disabled"
		statusColor := Error
		if t.Enabled {
			status = "enabled"
			statusColor = Success
		}

		globalPath := t.GlobalPath
		localPath := t.LocalPath

		row := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			nameWidth, t.Name,
			globalPathWidth, globalPath,
			localPathWidth, localPath,
			status)
		if UseColors() {
			b.WriteString(row[:len(row)-len(status)-1])
			b.WriteString(statusColor.Render(" " + status))
			b.WriteString("\n")
		} else {
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatTargetCompact formats targets in compact format.
func formatTargetCompact(targets []TargetOutput) string {
	var lines []string
	for _, t := range targets {
		lines = append(lines, t.Name)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// --- Skill List Formatting ---

// formatSkillTable formats skills as a table.
func formatSkillTable(skills []SkillOutput) string {
	if len(skills) == 0 {
		return "No skills installed."
	}

	// Sort skills: local first, then global, then by name
	sort.Slice(skills, func(i, j int) bool {
		// Extract scope from target (e.g., "pi/local" -> "local")
		scopeI := "global"
		scopeJ := "global"
		if strings.Contains(skills[i].Target, "/local") {
			scopeI = "local"
		}
		if strings.Contains(skills[j].Target, "/local") {
			scopeJ = "local"
		}

		// Local comes before global
		if scopeI != scopeJ {
			return scopeI == "local"
		}

		// Within same scope, sort by target then name
		if skills[i].Target != skills[j].Target {
			return skills[i].Target < skills[j].Target
		}
		return skills[i].Name < skills[j].Name
	})

	var b strings.Builder

	// Calculate column widths
	targetWidth := 10
	repoWidth := 10
	nameWidth := 10
	commitWidth := 7
	for _, s := range skills {
		if len(s.Target) > targetWidth {
			targetWidth = len(s.Target)
		}
		if len(s.Repo) > repoWidth {
			repoWidth = len(s.Repo)
		}
		if len(s.Name) > nameWidth {
			nameWidth = len(s.Name)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %*s",
		targetWidth, "TARGET",
		repoWidth, "REPO",
		nameWidth, "SKILL",
		commitWidth, "COMMIT")
	b.WriteString(header)
	b.WriteString("\n")

	// Print separator
	sep := strings.Repeat("─", targetWidth) + "  " +
		strings.Repeat("─", repoWidth) + "  " +
		strings.Repeat("─", nameWidth) + "  " +
		strings.Repeat("─", commitWidth)
	b.WriteString(sep)
	b.WriteString("\n")

	// Print rows
	for _, s := range skills {
		row := fmt.Sprintf("%-*s  %-*s  %-*s  %*s",
			targetWidth, s.Target,
			repoWidth, s.Repo,
			nameWidth, s.Name,
			commitWidth, s.Commit)
		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// formatSkillCompact formats skills in compact format.
func formatSkillCompact(skills []SkillOutput) string {
	// Sort: local first, then global, then by name
	sort.Slice(skills, func(i, j int) bool {
		scopeI := "global"
		scopeJ := "global"
		if strings.Contains(skills[i].Target, "/local") {
			scopeI = "local"
		}
		if strings.Contains(skills[j].Target, "/local") {
			scopeJ = "local"
		}
		if scopeI != scopeJ {
			return scopeI == "local"
		}
		if skills[i].Target != skills[j].Target {
			return skills[i].Target < skills[j].Target
		}
		return skills[i].Name < skills[j].Name
	})

	var lines []string
	for _, s := range skills {
		repoName := s.Repo
		if repoName == "" {
			repoName = "-"
		}
		lines = append(lines, fmt.Sprintf("%s/%s@%s repo=%s", s.Target, s.Name, s.Commit, repoName))
	}
	return strings.Join(lines, "\n")
}

// --- Repo List Formatting ---

// formatRepoTable formats repos as a table.
func formatRepoTable(repos []RepoOutput) string {
	if len(repos) == 0 {
		return "No repositories cached."
	}

	// Sort repos by name
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	var b strings.Builder

	// Calculate column widths
	nameWidth := 10
	urlWidth := 30
	for _, r := range repos {
		displayName := r.Name
		if r.Alias != "" {
			displayName = r.Alias
		}
		if len(displayName) > nameWidth {
			nameWidth = len(displayName)
		}
		if len(r.URL) > urlWidth {
			urlWidth = len(r.URL)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %-*s  %s",
		nameWidth, "REPO",
		urlWidth, "URL",
		"SKILLS")
	b.WriteString(header)
	b.WriteString("\n")

	// Print separator
	sep := strings.Repeat("─", nameWidth) + "  " +
		strings.Repeat("─", urlWidth) + "  " +
		"───────"
	b.WriteString(sep)
	b.WriteString("\n")

	// Print rows
	for _, r := range repos {
		displayName := r.Name
		if r.Alias != "" {
			displayName = r.Alias
		}
		row := fmt.Sprintf("%-*s  %-*s  %d",
			nameWidth, displayName,
			urlWidth, r.URL,
			r.SkillCount)
		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// formatRepoCompact formats repos in compact format.
func formatRepoCompact(repos []RepoOutput) string {
	var lines []string
	for _, r := range repos {
		displayName := r.Name
		if r.Alias != "" {
			displayName = r.Alias
		}
		lines = append(lines, fmt.Sprintf("%s (%d skills)", displayName, r.SkillCount))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
