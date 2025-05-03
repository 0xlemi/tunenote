# MacNote: Development Plan

## Phase 1: Foundation & Audio Capture

### Set up project structure and dependencies
- [x] Initialize Go module
- [x] Add core libraries (PortAudio, FFT)
- [x] Create basic project structure

### Create basic CLI framework
- [x] Set up basic application structure
- [ ] Set up command-line flags and options
- [ ] Create configuration handling
- [ ] Add logging infrastructure

### Implement basic audio capture
- [x] Create placeholder audio capture interface
- [x] Set up microphone input via PortAudio
- [x] Create audio buffer management
- [x] Implement simple audio level detection
- [ ] Test microphone access and raw audio capture

## Phase 2: Audio Analysis

### Implement frequency analysis
- [x] Add FFT processing for captured audio
- [x] Apply windowing function (Hann window)
- [x] Identify fundamental frequency

### Develop pitch detection algorithm
- [x] Implement note and octave calculation
- [x] Create frequency-to-note conversion
- [x] Add cents deviation calculation
- [ ] Test with various tones for accuracy

## Phase 3: Terminal UI Basics

### Create minimal viable UI
- [x] Set up Bubbletea framework
- [x] Display current note, octave, and frequency
- [x] Add basic styling with Lipgloss
- [x] Ensure UI updates don't block audio processing

## Phase 4: Refinement & Features

### Improve pitch detection accuracy
- [x] Fine-tune algorithms based on testing
- [x] Add noise filtering
- [x] Handle edge cases (very high/low notes)

### Enhance terminal UI
- [x] Add visualization of pitch accuracy
- [x] Implement note stability tracking
- [ ] Create settings menu for configuration
- [x] Polish visual design with rounded borders and color-coded notes

## Phase 5: Optimization & Completion

### Performance optimization
- [ ] Profile CPU and memory usage
- [ ] Optimize critical paths
- [ ] Reduce latency between sound and display

### Finalize and package
- [ ] Add comprehensive error handling
- [ ] Create user documentation
- [ ] Prepare for distribution (README, etc.) 