package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/0xlemi/tunenote/internal/audio"
	"github.com/0xlemi/tunenote/internal/pitch"
	"github.com/0xlemi/tunenote/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Audio settings
	bufferSize = 4096
	sampleRate = 44100
	channels   = 1

	// Debug settings
	enableLevelDebug = true                   // Set to true to update debug info in UI
	debugInterval    = time.Millisecond * 200 // How often to update debug info

	// Note stabilization
	stabilizationDelay = 300 * time.Millisecond // Delay after volume increase before registering note

	amplificationLevel = 8.0
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
	fmt.Println("TuneNote - Starting application...")

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
	isVolumeRising := false
	volumeRiseTime := time.Time{}
	lastDB := float32(-100)

	// Increase audio input sensitivity
	capturer.SetAmplification(amplificationLevel)

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

			// Send audio levels to UI instead of printing to terminal
			if enableLevelDebug && time.Since(lastDebugTime) > debugInterval {
				p.Send(ui.UpdateAudioLevelMsg{
					RMS: rms,
					DB:  db,
				})
				lastDebugTime = time.Now()
			}

			// Detect when volume is rising (note beginning)
			if db > lastDB+3 && db > -40 {
				// Volume is rising significantly and above threshold
				if !isVolumeRising {
					isVolumeRising = true
					volumeRiseTime = time.Now()
					// Don't attempt pitch detection until stabilization period is over
					lastDB = db
					time.Sleep(time.Millisecond * 10)
					continue
				}
			}
			lastDB = db

			// MUCH more aggressive silence detection - higher dB threshold
			// and clear notes immediately on silence
			if db < -30 { // Was -50, now -30 for more aggressive silence detection
				p.Send(ui.ClearNoteMsg{})
				isVolumeRising = false // Reset volume rising flag
				time.Sleep(time.Millisecond * 50)
				continue
			}

			// If we're in the initial rising volume period, wait for stabilization
			if isVolumeRising && time.Since(volumeRiseTime) < stabilizationDelay {
				time.Sleep(time.Millisecond * 10)
				continue
			}

			// Past stabilization period, note should be stable
			isVolumeRising = false

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
