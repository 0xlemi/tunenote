package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xlemi/tunenote/internal/pitch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Constants for UI behavior
const (
	// Timeline settings
	maxTimelineEntries = 50 // Maximum entries in the timeline
	timelineWidth      = 70 // Total width of the timeline
	noteDisplayWidth   = 3  // Width of each note entry in timeline
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

	debugStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	noSoundStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#888888")).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(2, 4).
			MarginBottom(1)

	timelineStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(0, 1).
			MarginTop(1).
			Width(timelineWidth + 4) // Add padding for borders

	timelineLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC"))

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#555555")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#999999")).
			Padding(0, 2).
			MarginLeft(2).
			Bold(true)

	clearButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#AA3333")). // Red background for clear button
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#662222")).
				Padding(0, 2).
				MarginLeft(2).
				Bold(true)

	// Standard box size
	boxWidth = 8

	// Note colors (moderate, not too bright, not too pastel)
	noteColors = map[string]string{
		"C": "#e5cf9e", // Moderate Beige
		"D": "#663e7d", // Medium Purple
		"E": "#e3a53e", // Moderate Yellow
		"F": "#c4563f", // Moderate Orange-Red
		"G": "#43873c", // Moderate Green
		"A": "#b64040", // Moderate Red
		"B": "#2a7bba", // Moderate Blue
	}
)

// TimelineEntry represents a note in the timeline with timestamp
type TimelineEntry struct {
	Note      *pitch.Note
	Timestamp time.Time
}

// Returns a style for a note
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

// Model represents the UI state
type Model struct {
	currentNote    *pitch.Note
	timeline       []TimelineEntry // Timeline of recent notes
	lastUpdate     time.Time
	width          int
	height         int
	isSilence      bool      // Whether we're currently detecting silence
	silenceSince   time.Time // When we first detected silence
	audioRMS       float32   // Current RMS level
	audioDB        float32   // Current dB level
	showDebug      bool      // Whether to show debug info
	timelineFrozen bool      // Whether the timeline is frozen/paused
}

// NewModel creates a new UI model
func NewModel() Model {
	return Model{
		currentNote:    nil,
		timeline:       make([]TimelineEntry, 0, maxTimelineEntries),
		lastUpdate:     time.Now(),
		isSilence:      true,
		silenceSince:   time.Now(),
		showDebug:      true, // Default to showing debug info
		timelineFrozen: false,
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

// UpdateAudioLevelMsg is a message to update the audio level display
type UpdateAudioLevelMsg struct {
	RMS float32
	DB  float32
}

// ClearNoteMsg is sent when we should clear the note display (no sound detected)
type ClearNoteMsg struct{}

// Update handles the model update based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "d":
			// Toggle debug display
			m.showDebug = !m.showDebug
		case "f", "space":
			// Toggle timeline freeze
			m.timelineFrozen = !m.timelineFrozen
		case "c":
			// Clear timeline history
			m.timeline = make([]TimelineEntry, 0, maxTimelineEntries)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		// Just keep the ticker running
		return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case UpdateNoteMsg:
		// We have a note, so we're not in silence mode
		m.isSilence = false
		note := pitch.Note(msg)

		// Check if the current note is different from the last note
		addToTimeline := true
		if m.currentNote != nil && note.Name == m.currentNote.Name && note.Octave == m.currentNote.Octave {
			// Same note as current, don't add to timeline
			addToTimeline = false
		}

		// Update current note
		m.currentNote = &note

		// Add to timeline if it's a new note and timeline is not frozen
		if addToTimeline && !m.timelineFrozen {
			// Create a copy to store in timeline
			noteCopy := note

			// Add to the end of the timeline
			entry := TimelineEntry{
				Note:      &noteCopy,
				Timestamp: time.Now(),
			}
			m.timeline = append(m.timeline, entry)

			// Trim timeline if it gets too long
			if len(m.timeline) > maxTimelineEntries {
				m.timeline = m.timeline[len(m.timeline)-maxTimelineEntries:]
			}
		}

		m.lastUpdate = time.Now()

	case UpdateAudioLevelMsg:
		// Update audio levels for display
		m.audioRMS = msg.RMS
		m.audioDB = msg.DB

	case ClearNoteMsg:
		// Immediately clear the note display - no delay
		m.currentNote = nil
		m.isSilence = true
		m.silenceSince = time.Now()
	}

	return m, nil
}

// getNextNote returns the next note in the scale (C -> D, D -> E, etc.)
func getNextNote(note string) string {
	noteOrder := []string{"C", "D", "E", "F", "G", "A", "B"}
	for i, n := range noteOrder {
		if n == note {
			if i < len(noteOrder)-1 {
				return noteOrder[i+1]
			}
			return noteOrder[0] // Wrap around to C if B
		}
	}
	return note // Fallback
}

// getNoteColor returns the color for a note
func getNoteColor(noteName string) string {
	if strings.HasSuffix(noteName, "#") {
		// For sharp notes, use the base note color
		baseNote := string(noteName[0])
		return noteColors[baseNote]
	}
	return noteColors[noteName]
}

// renderTimelineNote renders a compact note representation for the timeline
func renderTimelineNote(note *pitch.Note) string {
	if note == nil {
		return strings.Repeat(" ", noteDisplayWidth)
	}

	// Create a compact representation of the note (e.g., "C4", "D#5")
	noteText := note.Name
	if len(noteText) == 1 {
		noteText += " " // Add space for single-char notes to align with sharps
	}

	// Create style with appropriate color
	noteColor := getNoteColor(note.Name)
	timelineNoteStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(noteColor)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(noteDisplayWidth).
		Align(lipgloss.Center)

	return timelineNoteStyle.Render(noteText)
}

// View renders the UI
func (m Model) View() string {
	s := titleStyle.Render("TuneNote - Musical Note Detector")
	s += "\n"

	if m.currentNote != nil {
		// Get note style based on the note name
		noteStyle := getNoteStyle(m.currentNote.Name)

		// Generate note text
		noteText := fmt.Sprintf("%s%d", m.currentNote.Name, m.currentNote.Octave)

		// For sharps, we need to render the note with split colors
		if strings.HasSuffix(m.currentNote.Name, "#") {
			baseNote := string(m.currentNote.Name[0])
			nextNote := getNextNote(baseNote)

			baseColor := noteColors[baseNote]
			nextColor := noteColors[nextNote]

			// Create joined style with rounded border
			joinedStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#333333")).
				Padding(2, 4).
				Width(boxWidth / 2). // Half width
				Align(lipgloss.Center).
				MarginBottom(1)

			// Split rendering approach for sharp notes
			baseStyle := joinedStyle.Copy().Background(lipgloss.Color(baseColor))
			sharpStyle := joinedStyle.Copy().Background(lipgloss.Color(nextColor))

			// Render each part separately
			baseChar := string(noteText[0])
			sharpChar := "#"
			octave := noteText[2:]

			// Combine the parts
			s += lipgloss.JoinHorizontal(lipgloss.Top,
				baseStyle.Render(baseChar),
				sharpStyle.Render(sharpChar+octave))

		} else {
			// For natural notes, use a single color with fixed width
			noteStyle = noteStyle.Width(boxWidth).Align(lipgloss.Center)
			s += noteStyle.Render(noteText)
		}

		s += "\n"

		info := fmt.Sprintf("Frequency: %.2f Hz | Cents: %+.1f",
			m.currentNote.Frequency,
			m.currentNote.Cents)
		s += infoStyle.Render(info)
	} else {
		// No note being detected - show gray placeholder box
		placeholder := noSoundStyle.Width(boxWidth).Align(lipgloss.Center).Render("---")
		s += placeholder
		s += "\n"
		s += infoStyle.Render("Make a sound to see the note...")
	}

	s += "\n"

	// Render timeline
	if len(m.timeline) > 0 {
		// Create timeline header with freeze button
		var timelineHeader string
		freezeButtonText := "Freeze"
		if m.timelineFrozen {
			freezeButtonText = "Resume"
			timelineHeader = timelineLabelStyle.Render("Timeline: FROZEN")
		} else {
			timelineHeader = timelineLabelStyle.Render("Timeline: (newest notes on the right)")
		}

		// Add the freeze/resume button
		freezeButton := buttonStyle.Render(freezeButtonText)

		// Add clear button
		clearButton := clearButtonStyle.Render("Clear")

		// Join all header elements
		timelineHeader = lipgloss.JoinHorizontal(lipgloss.Top, timelineHeader, freezeButton, clearButton)

		s += timelineHeader
		s += "\n"

		// Create timeline display
		timelineContent := ""

		// Calculate how many entries we can show in the timeline
		entriesToShow := len(m.timeline)
		startIndex := 0

		if entriesToShow > timelineWidth/noteDisplayWidth {
			entriesToShow = timelineWidth / noteDisplayWidth
			startIndex = len(m.timeline) - entriesToShow
		}

		// Create the timeline as a series of colored blocks
		for i := startIndex; i < len(m.timeline); i++ {
			timelineContent += renderTimelineNote(m.timeline[i].Note)
		}

		// Wrap it in the timeline box
		s += timelineStyle.Render(timelineContent)
		s += "\n"
	} else {
		// Show empty timeline box
		emptyMessage := "No notes recorded yet"
		s += timelineStyle.Render(emptyMessage)
	}

	// Show debug info if enabled
	if m.showDebug {
		dbInfo := fmt.Sprintf("Audio Level: RMS=%.6f, dB=%.1f", m.audioRMS, m.audioDB)
		s += debugStyle.Render(dbInfo)
		s += "\n"
	}

	s += "\n"
	s += infoStyle.Render("Press f or space to freeze/resume | Press c to clear history | Press d to toggle debug | Press q to quit")

	return s
}
