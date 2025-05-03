# TuneNote

A note-taking application designed for macOS that detects musical notes from audio input. TuneNote captures audio from your microphone, analyzes the frequency, and displays the corresponding musical note in real-time through a terminal UI.

## Features

- Real-time audio capture and analysis
- Accurate pitch detection with cents deviation
- Musical note and octave identification
- Terminal-based UI with note visualization
- Low latency performance

## Development

See [PLAN.md](PLAN.md) for the detailed development roadmap.

## Prerequisites
- Go 1.19+
- PortAudio development libraries
- FFT library

## Getting Started
1. Install dependencies
```bash
# macOS
brew install portaudio

# Go dependencies
go mod tidy
```

2. Build the application
```bash
go build -o tunenote ./cmd
```

3. Run the application
```bash
./tunenote
``` 