package pitch

import (
	"errors"
	"math"

	"github.com/0xlemi/tunenote/internal/audio"
)

// Errors
var (
	ErrEmptyBuffer     = errors.New("empty audio buffer")
	ErrVolumeThreshold = errors.New("volume below threshold")
)

// Note represents a musical note
type Note struct {
	Name      string  // e.g., "A", "A#", "B"
	Octave    int     // e.g., 4 for middle C (C4)
	Frequency float64 // Frequency in Hz
	Cents     float64 // Cents deviation from perfect pitch (-50 to +50)
}

// Detector defines the interface for pitch detection
type Detector interface {
	// DetectPitch analyzes an audio buffer and returns the detected note
	DetectPitch(buffer *audio.AudioBuffer) (*Note, error)
}

// DefaultDetector is a basic implementation of pitch detection
type DefaultDetector struct{}

// NewDefaultDetector creates a new pitch detector
func NewDefaultDetector() *DefaultDetector {
	return &DefaultDetector{}
}

// Musical note frequencies (A4 = 440Hz)
var noteFrequencies = map[string]float64{
	"C":  16.35,
	"C#": 17.32,
	"D":  18.35,
	"D#": 19.45,
	"E":  20.60,
	"F":  21.83,
	"F#": 23.12,
	"G":  24.50,
	"G#": 25.96,
	"A":  27.50,
	"A#": 29.14,
	"B":  30.87,
}

// All note names in chromatic order
var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// DetectPitch analyzes an audio buffer and returns the detected note
func (d *DefaultDetector) DetectPitch(buffer *audio.AudioBuffer) (*Note, error) {
	if buffer == nil || len(buffer.Samples) == 0 {
		return nil, errors.New("empty audio buffer")
	}

	// TODO: Implement actual pitch detection using FFT
	// This is a placeholder that returns A4 (440Hz)
	frequency := 440.0

	return frequencyToNote(frequency), nil
}

// frequencyToNote converts a frequency to a musical note
func frequencyToNote(frequency float64) *Note {
	// A4 = 440Hz, calculate semitones from A4
	semitones := 12 * math.Log2(frequency/440.0)

	// Round to nearest semitone
	roundedSemitones := math.Round(semitones)

	// Calculate cents deviation (difference between actual and rounded semitones)
	cents := 100 * (semitones - roundedSemitones)

	// Calculate note index (0 = C, 1 = C#, etc.)
	// A4 is 9 semitones above C4, so we add 9 to the semitone count
	noteIndex := int(math.Mod(roundedSemitones+9, 12))
	if noteIndex < 0 {
		noteIndex += 12
	}

	// Calculate octave (A4 is in octave 4)
	octave := 4 + int(math.Floor((roundedSemitones+9)/12))

	// Get note name
	noteName := noteNames[noteIndex]

	return &Note{
		Name:      noteName,
		Octave:    octave,
		Frequency: frequency,
		Cents:     cents,
	}
}
