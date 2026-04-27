package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// outputFormat represents the output format.
type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
)

// parseFormat parses the format flag value.
func parseFormat(s string) outputFormat {
	switch strings.ToLower(s) {
	case "json", "js":
		return formatJSON
	default:
		return formatText
	}
}

// printJSON prints data as formatted JSON to stdout.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// TargetOutput represents target list output for JSON.
type TargetOutput struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

// RepoOutput represents repo list output for JSON.
type RepoOutput struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Branch   string `json:"branch"`
	SkillCount int   `json:"skill_count"`
	Updated  string `json:"updated"`
}

// SkillOutput represents skill list output for JSON.
type SkillOutput struct {
	Name    string `json:"name"`
	Source  string `json:"source,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Target  string `json:"target,omitempty"`
	Description string `json:"description,omitempty"`
}

// SearchResultOutput represents search results for JSON.
type SearchResultOutput struct {
	Query   string        `json:"query"`
	Results []SkillOutput `json:"results"`
	Count   int           `json:"count"`
}

// ErrorOutput represents error output for JSON.
type ErrorOutput struct {
	Error   string `json:"error"`
	Command string `json:"command,omitempty"`
}

// printError prints an error message.
func printError(format string, args ...interface{}) {
	if parseFormat(formatFlag) == formatJSON {
		printJSON(ErrorOutput{Error: fmt.Sprintf(format, args...)})
	} else {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
