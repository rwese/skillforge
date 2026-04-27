package main

import (
	"fmt"
	"time"
)

// Spinner displays a simple spinner animation.
type Spinner struct {
	message string
	done    chan struct{}
}

// NewSpinner creates a new spinner with a message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation in a goroutine.
func (s *Spinner) Start() {
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r%s %s", frames[i%len(frames)], s.message)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
		fmt.Printf("\r\x1b[2K") // Clear line
	}
}

// Messagef updates the spinner message.
func (s *Spinner) Messagef(format string, args ...interface{}) {
	s.message = fmt.Sprintf(format, args...)
}

// Progress represents a progress tracker.
type Progress struct {
	total    int
	current  int
	message  string
	filename string
}

// NewProgress creates a new progress tracker.
func NewProgress(total int, message string) *Progress {
	return &Progress{
		total:   total,
		current: 0,
		message: message,
	}
}

// Increment increases the current count by 1.
func (p *Progress) Increment() {
	p.current++
	p.draw()
}

// SetFile sets the current filename being processed.
func (p *Progress) SetFile(filename string) {
	p.filename = filename
	p.draw()
}

// draw renders the progress.
func (p *Progress) draw() {
	if p.total > 0 {
		percent := int(float64(p.current) / float64(p.total) * 100)
		fmt.Printf("\r%s [%d/%d] %d%% %s", p.message, p.current, p.total, percent, p.filename)
	} else {
		fmt.Printf("\r%s %s", p.message, p.filename)
	}
}

// Complete finishes the progress.
func (p *Progress) Complete() {
	p.current = p.total
	p.draw()
	fmt.Println()
}
