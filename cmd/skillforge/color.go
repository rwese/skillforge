package main

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Color definitions using lipgloss.
var (
	// Success represents success messages (green checkmark).
	Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2ECC71"))

	// Error represents error messages (red).
	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E74C3C"))

	// Warning represents warning messages (yellow/amber).
	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F39C12"))

	// Info represents info/redirect messages (cyan).
	Info = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB"))

	// Highlight represents highlighted text (bold).
	Highlight = lipgloss.NewStyle().
			Bold(true)

	// Dim represents dimmed/secondary text.
	Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D"))

	// TableHeader represents table header cells.
	TableHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")).
			Bold(true)

	// TableCell represents table cell content.
	TableCell = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECF0F1"))
)

// Checkmark returns a styled checkmark string.
func Checkmark() string {
	if !UseColors() {
		return "✓"
	}
	return Success.Render("✓")
}

// XMark returns a styled X mark string.
func XMark() string {
	if !UseColors() {
		return "✗"
	}
	return Error.Render("✗")
}

// WarningMark returns a styled warning mark string.
func WarningMark() string {
	if !UseColors() {
		return "!"
	}
	return Warning.Render("!")
}

// Arrow returns a styled arrow string.
func Arrow() string {
	if !UseColors() {
		return "→"
	}
	return Info.Render("→")
}

// Dot returns a styled bullet/dot string.
func Dot() string {
	return "•"
}

// UseColors determines whether to use colored output.
func UseColors() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	// Check if stdout is a terminal
	return isTerminal()
}

// isTerminal checks if stdout is a terminal.
func isTerminal() bool {
	// Simple check: if NO_COLOR is not set and TERM is set to something
	// that isn't dumb, assume we're in a terminal.
	// For more robust detection, you could use syscall.IsTerminal,
	// but this is sufficient for most cases.
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (TTY)
	mode := fileInfo.Mode()
	return mode&os.ModeCharDevice != 0
}

// Styles for formatted output.
var (
	// BoxStyle is a bordered box style.
	BoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1)

	// SuccessBox is a green bordered box.
	SuccessBox = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#2ECC71")).
			Padding(1)

	// ErrorBox is a red bordered box.
	ErrorBox = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#E74C3C")).
			Padding(1)

	// WarningBox is a yellow bordered box.
	WarningBox = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#F39C12")).
			Padding(1)
)
