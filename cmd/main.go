package main

import (
	"fmt"
	"log"
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
)

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

	// Start a goroutine for audio processing
	go func() {
		for {
			// Get audio buffer
			buffer, err := capturer.GetBuffer()
			if err != nil {
				continue
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

			// Sleep a bit to avoid excessive CPU usage
			time.Sleep(time.Millisecond * 50)
		}
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
