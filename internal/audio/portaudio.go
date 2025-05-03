package audio

import (
	"errors"
	"sync"

	"github.com/gordonklaus/portaudio"
)

// PortAudioCapturer implements audio capture using PortAudio
type PortAudioCapturer struct {
	isCapturing   bool
	stream        *portaudio.Stream
	buffer        *AudioBuffer
	bufferSize    int
	sampleRate    int
	channels      int
	inputBuffer   []float32
	bufferMutex   sync.Mutex
	amplification float32 // Audio signal amplification factor
}

// NewPortAudioCapturer creates a new audio capturer using PortAudio
func NewPortAudioCapturer(bufferSize, sampleRate, channels int) (*PortAudioCapturer, error) {
	// Initialize PortAudio
	err := portaudio.Initialize()
	if err != nil {
		return nil, err
	}

	capturer := &PortAudioCapturer{
		isCapturing: false,
		buffer: &AudioBuffer{
			Samples:    make([]float32, 0, bufferSize),
			SampleRate: sampleRate,
		},
		bufferSize:    bufferSize,
		sampleRate:    sampleRate,
		channels:      channels,
		inputBuffer:   make([]float32, bufferSize*channels),
		amplification: 5.0, // Amplify input signal by 5x
	}

	return capturer, nil
}

// Start begins audio capture
func (c *PortAudioCapturer) Start() error {
	if c.isCapturing {
		return errors.New("audio capture already started")
	}

	// Open default input stream
	var err error
	c.stream, err = portaudio.OpenDefaultStream(
		c.channels, // input channels
		0,          // output channels (we don't need output)
		float64(c.sampleRate),
		c.bufferSize/c.channels, // frames per buffer
		c.processAudio,          // callback function
	)
	if err != nil {
		return err
	}

	// Start the stream
	err = c.stream.Start()
	if err != nil {
		c.stream.Close()
		return err
	}

	c.isCapturing = true
	return nil
}

// Stop ends audio capture
func (c *PortAudioCapturer) Stop() error {
	if !c.isCapturing {
		return errors.New("audio capture not started")
	}

	// Stop and close the stream
	err := c.stream.Stop()
	if err != nil {
		return err
	}

	err = c.stream.Close()
	if err != nil {
		return err
	}

	// Terminate PortAudio
	err = portaudio.Terminate()
	if err != nil {
		return err
	}

	c.isCapturing = false
	return nil
}

// processAudio is the callback function for audio processing
func (c *PortAudioCapturer) processAudio(in, _ []float32) {
	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()

	// If we have multi-channel input, we'll average the channels
	if c.channels > 1 {
		// Create a mono buffer for averaging channels
		monoBuffer := make([]float32, len(in)/c.channels)

		// Average each set of channel samples and apply amplification
		for i := 0; i < len(monoBuffer); i++ {
			sum := float32(0)
			for ch := 0; ch < c.channels; ch++ {
				sum += in[i*c.channels+ch]
			}
			// Average the channels and apply amplification
			monoBuffer[i] = (sum / float32(c.channels)) * c.amplification
		}

		// Update the buffer
		c.buffer.Samples = monoBuffer
	} else {
		// Just copy the mono input and apply amplification
		c.buffer.Samples = make([]float32, len(in))
		for i, sample := range in {
			c.buffer.Samples[i] = sample * c.amplification
		}
	}
}

// GetBuffer returns the current audio buffer
func (c *PortAudioCapturer) GetBuffer() (*AudioBuffer, error) {
	if !c.isCapturing {
		return nil, errors.New("audio capture not started")
	}

	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()

	// Create a copy of the buffer to return
	bufferCopy := &AudioBuffer{
		Samples:    make([]float32, len(c.buffer.Samples)),
		SampleRate: c.buffer.SampleRate,
	}
	copy(bufferCopy.Samples, c.buffer.Samples)

	return bufferCopy, nil
}

// IsCapturing returns true if currently capturing audio
func (c *PortAudioCapturer) IsCapturing() bool {
	return c.isCapturing
}

// SetAmplification sets the audio amplification factor
func (c *PortAudioCapturer) SetAmplification(factor float32) {
	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()

	// Ensure amplification is positive
	if factor < 0.1 {
		factor = 0.1
	}

	c.amplification = factor
}
