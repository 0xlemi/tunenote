package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xlemi/macnote/internal/pitch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Constants for UI behavior
const (
	// How long a note needs to be present to be considered stable (milliseconds)
	noteStabilityThreshold = 300

	// How long to keep displaying a stable note after it changes (milliseconds)
	noteDisplayDuration = 500
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	// Note colors
	noteColors = map[string]string{
		"C": "#E8D6B0", // Beige
		"D": "#A020F0", // Purple
		"E": "#FFFF00", // Yellow
		"F": "#FFA500", // Orange
		"G": "#00FF00", // Green
		"A": "#FF0000", // Red
		"B": "#0000FF", // Blue
	}
)

// Returns a style for a note (including sharps which get split color)
func getNoteStyle(noteName string) lipgloss.Style {
	if strings.HasSuffix(noteName, "#") {
		// For sharp notes, we handle the rendering separately in View()
		// Just return a basic style
		return lipgloss.NewStyle().Bold(true).MarginBottom(1)
	} else {
		// For natural notes, use a single color
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color(noteColors[noteName])).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(2, 4).
			MarginBottom(1)
	}
}

// Get the next note in the scale (for sharp note colors)
func getNextNote(note string) string {
	switch note {
	case "C":
		return "D"
	case "D":
		return "E"
	case "E":
		return "F"
	case "F":
		return "G"
	case "G":
		return "A"
	case "A":
		return "B"
	case "B":
		return "C"
	default:
		return "C"
	}
}

// Model represents the UI state
type Model struct {
	currentNote    *pitch.Note
	stableNote     *pitch.Note
	notesHistory   map[string]time.Time // Track when we first saw each note
	stableNoteTime time.Time            // When the stable note was set
	lastNoteName   string               // Last note we displayed
	lastUpdated    time.Time
	updateTicker   time.Time
	width          int
	height         int
}

// NewModel creates a new UI model
func NewModel() Model {
	return Model{
		currentNote:    nil,
		stableNote:     nil,
		notesHistory:   make(map[string]time.Time),
		stableNoteTime: time.Time{},
		lastNoteName:   "",
		lastUpdated:    time.Now(),
		updateTicker:   time.Now(),
	}
}

// Init initializes the UI model
func (m Model) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// TickMsg represents a timer tick
type TickMsg time.Time

// UpdateNoteMsg is a message to update the current note
type UpdateNoteMsg pitch.Note

// Update updates the UI model based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		m.updateTicker = time.Time(msg)

		// Clean up old notes from history (older than 2 seconds)
		now := time.Now()
		for note, timestamp := range m.notesHistory {
			if now.Sub(timestamp) > 2*time.Second {
				delete(m.notesHistory, note)
			}
		}

		return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case UpdateNoteMsg:
		note := pitch.Note(msg)
		m.currentNote = &note

		// Note stability logic
		noteName := fmt.Sprintf("%s%d", note.Name, note.Octave)

		// Record when we first saw this note
		if _, exists := m.notesHistory[noteName]; !exists {
			m.notesHistory[noteName] = time.Now()
		}

		// Check if this note has been present long enough to be stable
		noteFirstSeen := m.notesHistory[noteName]
		if time.Since(noteFirstSeen) >= noteStabilityThreshold*time.Millisecond {
			// This note is stable, update the stable note
			m.stableNote = &note
			m.stableNoteTime = time.Now()
			m.lastNoteName = noteName
		} else if m.stableNote != nil {
			// Check if we should still show the previous stable note
			if time.Since(m.stableNoteTime) < noteDisplayDuration*time.Millisecond {
				// Keep the stable note for display continuity
			} else {
				// Time to update to a new stable note if one exists
				// Check if any note has been stable long enough
				var longestNote string
				var longestTime time.Duration

				for name, firstSeen := range m.notesHistory {
					duration := time.Since(firstSeen)
					if duration > longestTime && duration >= noteStabilityThreshold*time.Millisecond {
						longestNote = name
						longestTime = duration
					}
				}

				// If we found a stable note, update
				if longestNote != "" && longestNote != m.lastNoteName {
					// The new note is now considered stable
					m.lastNoteName = longestNote
					// We need to reconstruct the note from the name
					parts := strings.Split(longestNote, "")
					name := parts[0]
					if len(parts) > 2 && parts[1] == "#" {
						name += "#"
					}
					octave := 4 // Default to middle octave if parsing fails
					fmt.Sscanf(longestNote[len(name):], "%d", &octave)

					// Update the stable note
					if m.currentNote != nil && m.currentNote.Name == name {
						m.stableNote = m.currentNote
					}
					m.stableNoteTime = time.Now()
				}
			}
		}

		m.lastUpdated = time.Now()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	s := titleStyle.Render("MacNote - Musical Note Detector")
	s += "\n"

	// Determine which note to display (stable or current)
	noteToDisplay := m.stableNote
	if noteToDisplay == nil {
		noteToDisplay = m.currentNote
	}

	if noteToDisplay != nil {
		// Get note style based on the note name
		noteStyle := getNoteStyle(noteToDisplay.Name)

		// Generate note text
		noteText := fmt.Sprintf("%s%d", noteToDisplay.Name, noteToDisplay.Octave)

		// For sharps, we need to render the note with split colors
		if strings.HasSuffix(noteToDisplay.Name, "#") {
			baseNote := string(noteToDisplay.Name[0])
			nextNote := getNextNote(baseNote)

			baseColor := noteColors[baseNote]
			nextColor := noteColors[nextNote]

			// Create left and right styles with rounded borders
			leftStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color(baseColor)).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#333333")).
				BorderLeft(true).
				BorderTop(true).
				BorderBottom(true).
				BorderRight(false).
				PaddingLeft(2).
				PaddingRight(1).
				PaddingTop(2).
				PaddingBottom(2)

			rightStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color(nextColor)).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#333333")).
				BorderLeft(false).
				BorderTop(true).
				BorderBottom(true).
				BorderRight(true).
				PaddingLeft(1).
				PaddingRight(2).
				PaddingTop(2).
				PaddingBottom(2)

			// Render the note with split colors
			baseNoteChar := string(noteText[0])
			sharpChar := "#"
			octave := noteText[2:]

			s += leftStyle.Render(baseNoteChar) + rightStyle.Render(sharpChar+octave)
		} else {
			// For natural notes, use a single color
			s += noteStyle.Render(noteText)
		}

		s += "\n"

		info := fmt.Sprintf("Frequency: %.2f Hz | Cents: %+.1f",
			noteToDisplay.Frequency,
			noteToDisplay.Cents)
		s += infoStyle.Render(info)
	} else {
		s += infoStyle.Render("Listening for audio...")
	}

	s += "\n\n"
	s += infoStyle.Render("Press q to quit")

	return s
}
