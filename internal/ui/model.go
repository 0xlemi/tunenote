package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xlemi/macnote/internal/pitch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	currentNote  *pitch.Note
	lastUpdated  time.Time
	updateTicker time.Time
	width        int
	height       int
}

// NewModel creates a new UI model
func NewModel() Model {
	return Model{
		currentNote:  nil,
		lastUpdated:  time.Now(),
		updateTicker: time.Now(),
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
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		m.updateTicker = time.Time(msg)
		return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case UpdateNoteMsg:
		note := pitch.Note(msg)
		m.currentNote = &note
		m.lastUpdated = time.Now()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	s := titleStyle.Render("MacNote - Musical Note Detector")
	s += "\n"

	if m.currentNote != nil {
		// Get note style based on the note name
		noteStyle := getNoteStyle(m.currentNote.Name)

		// For sharps, we need to render the note with split colors
		if strings.HasSuffix(m.currentNote.Name, "#") {
			baseNote := string(m.currentNote.Name[0])
			nextNote := getNextNote(baseNote)

			baseColor := noteColors[baseNote]
			nextColor := noteColors[nextNote]

			// Create left and right styles for the split color appearance
			leftStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color(baseColor)).
				PaddingLeft(2).
				PaddingRight(1)

			rightStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color(nextColor)).
				PaddingLeft(1).
				PaddingRight(2)

			// Render the note with split colors
			noteText := fmt.Sprintf("%s%d", m.currentNote.Name, m.currentNote.Octave)
			baseNoteChar := string(noteText[0])
			sharpChar := "#"
			octave := noteText[2:]

			s += leftStyle.Render(baseNoteChar) + rightStyle.Render(sharpChar+octave)
		} else {
			// For natural notes, use a single color
			noteText := fmt.Sprintf("%s%d", m.currentNote.Name, m.currentNote.Octave)
			s += noteStyle.Render(noteText)
		}

		s += "\n"

		info := fmt.Sprintf("Frequency: %.2f Hz | Cents: %+.1f",
			m.currentNote.Frequency,
			m.currentNote.Cents)
		s += infoStyle.Render(info)
	} else {
		s += infoStyle.Render("Listening for audio...")
	}

	s += "\n\n"
	s += infoStyle.Render("Press q to quit")

	return s
}
