package pitch

import (
	"math"
	"math/cmplx"

	"github.com/0xlemi/macnote/internal/audio"
	"github.com/mjibson/go-dsp/fft"
)

// FFTDetector implements pitch detection using FFT
type FFTDetector struct {
	windowSize int
}

// NewFFTDetector creates a new FFT-based pitch detector
func NewFFTDetector(windowSize int) *FFTDetector {
	return &FFTDetector{
		windowSize: windowSize,
	}
}

// DetectPitch analyzes an audio buffer and returns the detected note
func (d *FFTDetector) DetectPitch(buffer *audio.AudioBuffer) (*Note, error) {
	if buffer == nil || len(buffer.Samples) == 0 {
		return nil, ErrEmptyBuffer
	}

	// Apply windowing function (Hann window)
	windowedSamples := applyHannWindow(buffer.Samples)

	// Convert from []float32 to []complex128 for the FFT
	complexSamples := make([]complex128, len(windowedSamples))
	for i, sample := range windowedSamples {
		complexSamples[i] = complex(float64(sample), 0)
	}

	// Perform FFT
	spectrum := fft.FFT(complexSamples)

	// Find the peak frequency
	peakFreq := findPeakFrequency(spectrum, buffer.SampleRate)

	// Convert frequency to note
	return frequencyToNote(peakFreq), nil
}

// applyHannWindow applies a Hann window to the audio samples
func applyHannWindow(samples []float32) []float32 {
	windowedSamples := make([]float32, len(samples))
	for i, sample := range samples {
		// Hann window coefficient
		windowCoeff := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(samples)-1)))
		windowedSamples[i] = sample * float32(windowCoeff)
	}
	return windowedSamples
}

// findPeakFrequency finds the frequency with the highest magnitude in the spectrum
func findPeakFrequency(spectrum []complex128, sampleRate int) float64 {
	// We only need to look at the first half of the spectrum (Nyquist theorem)
	spectrumHalf := spectrum[:len(spectrum)/2]

	// Find the bin with the maximum magnitude
	maxMagnitude := 0.0
	maxBin := 0

	// Skip the first few bins to avoid DC component and very low frequencies
	for i := 3; i < len(spectrumHalf); i++ {
		magnitude := cmplx.Abs(spectrumHalf[i])
		if magnitude > maxMagnitude {
			maxMagnitude = magnitude
			maxBin = i
		}
	}

	// Convert bin index to frequency
	binSizeHz := float64(sampleRate) / float64(len(spectrum))
	return float64(maxBin) * binSizeHz
}
