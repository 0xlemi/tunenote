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

// getAudioLevel calculates RMS and peak audio levels
func getAudioLevel(buffer *audio.AudioBuffer) (rms, peak float32) {
	if buffer == nil || len(buffer.Samples) == 0 {
		return 0, 0
	}

	sumSquares := float32(0)
	peakVal := float32(0)

	for _, sample := range buffer.Samples {
		// Get absolute value
		absVal := float32(math.Abs(float64(sample)))

		// For peak
		if absVal > peakVal {
			peakVal = absVal
		}

		// For RMS
		sumSquares += sample * sample
	}

	rms = float32(math.Sqrt(float64(sumSquares / float32(len(buffer.Samples)))))
	return rms, peakVal
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

	// Variables for audio level debug
	lastDebugTime := time.Now()

	// Adjust the initial amplification
	capturer.SetAmplification(10.0) // Start with higher sensitivity

	// Start a goroutine for audio processing
	go func() {
		for {
			// Get audio buffer
			buffer, err := capturer.GetBuffer()
			if err != nil {
				continue
			}

			// Debug audio levels
			if enableLevelDebug && time.Since(lastDebugTime) > debugInterval {
				rms, peak := getAudioLevel(buffer)
				fmt.Printf("Audio levels - RMS: %.6f, Peak: %.6f\n", rms, peak)
				lastDebugTime = time.Now()
			}

			// Skip if buffer is empty or too small
			if len(buffer.Samples) < 512 {
				time.Sleep(time.Millisecond * 10)
				continue
			}

			// Detect pitch
			note, err := detector.DetectPitch(buffer)
			if err != nil {
				time.Sleep(time.Millisecond * 10)
				continue
			}

			// Update UI with detected note
			p.Send(ui.UpdateNoteMsg(*note))

			// Sleep a bit to avoid excessive CPU usage and reduce flickering
			time.Sleep(time.Millisecond * 100)
		}
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
