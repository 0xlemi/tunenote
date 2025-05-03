package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/0xlemi/macnote/internal/audio"
	"github.com/0xlemi/macnote/internal/pitch"
	"github.com/0xlemi/macnote/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Audio settings
	bufferSize = 4096
	sampleRate = 44100
	channels   = 1

	// Debug settings
	enableLevelDebug = true            // Set to true to print audio levels
	debugInterval    = time.Second * 2 // How often to print debug info
)

// getAudioLevel calculates RMS and dB level
func getAudioLevel(buffer *audio.AudioBuffer) (rms, db float32) {
	if buffer == nil || len(buffer.Samples) == 0 {
		return 0, -100
	}

	sumSquares := float32(0)

	for _, sample := range buffer.Samples {
		sumSquares += sample * sample
	}

	rms = float32(math.Sqrt(float64(sumSquares / float32(len(buffer.Samples)))))

	// Calculate dB (with protection against log(0))
	if rms > 0.0000001 { // Avoid log(0)
		// Convert to dB: dB = 20 * log10(amplitude)
		db = 20 * float32(math.Log10(float64(rms)))
	} else {
		db = -100
	}

	return rms, db
}

func main() {
	fmt.Println("MacNote - Starting application...")

	// Create audio capturer with PortAudio
	capturer, err := audio.NewPortAudioCapturer(bufferSize, sampleRate, channels)
	if err != nil {
		log.Fatalf("Failed to create audio capturer: %v", err)
	}

	// Create FFT-based pitch detector
	detector := pitch.NewFFTDetector(bufferSize)

	// Create UI model
	model := ui.NewModel()

	// Start audio capture
	err = capturer.Start()
	if err != nil {
		log.Fatalf("Failed to start audio capture: %v", err)
	}
	defer capturer.Stop()

	// Start UI
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Variables
	lastDebugTime := time.Now()
	lastNoteTime := time.Now()

	// Increase audio input sensitivity
	capturer.SetAmplification(7.0)

	// Print startup message
	fmt.Println("Listening for musical notes...")

	// Start a goroutine for audio processing
	go func() {
		for {
			// Get audio buffer
			buffer, err := capturer.GetBuffer()
			if err != nil {
				time.Sleep(time.Millisecond * 10)
				continue
			}

			// Skip if buffer is empty or too small
			if len(buffer.Samples) < 512 {
				time.Sleep(time.Millisecond * 10)
				continue
			}

			// Get audio levels for monitoring
			rms, db := getAudioLevel(buffer)

			// Debug output
			if enableLevelDebug && time.Since(lastDebugTime) > debugInterval {
				fmt.Printf("Audio: RMS=%.6f, dB=%.1f\n", rms, db)
				lastDebugTime = time.Now()
			}

			// MUCH more aggressive silence detection - higher dB threshold
			// and clear notes immediately on silence
			if db < -30 { // Was -50, now -30 for more aggressive silence detection
				p.Send(ui.ClearNoteMsg{})
				time.Sleep(time.Millisecond * 50)
				continue
			}

			// Try to detect pitch
			note, err := detector.DetectPitch(buffer)
			if err != nil {
				// Any error in pitch detection should clear the display
				p.Send(ui.ClearNoteMsg{})
				time.Sleep(time.Millisecond * 50)
				continue
			}

			// Only send note updates at reasonable intervals to prevent flicker
			if time.Since(lastNoteTime) > 80*time.Millisecond {
				p.Send(ui.UpdateNoteMsg(*note))
				lastNoteTime = time.Now()
			}

			// Sleep a bit to avoid excessive CPU usage
			time.Sleep(time.Millisecond * 50)
		}
	}()

	// Run the UI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
