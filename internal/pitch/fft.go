package pitch

import (
	"math"
	"math/cmplx"
	"sort"

	"github.com/0xlemi/macnote/internal/audio"
	"github.com/mjibson/go-dsp/fft"
)

// FFTDetector implements pitch detection using FFT
type FFTDetector struct {
	windowSize      int
	minFrequency    float64 // Lowest frequency to detect (Hz)
	maxFrequency    float64 // Highest frequency to detect (Hz)
	noiseFloor      float64 // Noise threshold (0.0-1.0)
	peakThreshold   float64 // Minimum peak height as fraction of highest peak
	volumeThreshold float64 // Minimum RMS volume level for note detection
}

// NewFFTDetector creates a new FFT-based pitch detector
func NewFFTDetector(windowSize int) *FFTDetector {
	return &FFTDetector{
		windowSize:      windowSize,
		minFrequency:    80.0,   // E2 on guitar is ~82 Hz
		maxFrequency:    1200.0, // E6 on guitar is ~1319 Hz
		noiseFloor:      0.01,   // Reduced from 0.05 to 0.01 (more sensitive to quieter sounds)
		peakThreshold:   0.2,    // Reduced from 0.3 to 0.2 (consider smaller peaks as valid)
		volumeThreshold: 0.005,  // Increased from 0.002 to 0.005 for better silence handling
	}
}

// DetectPitch analyzes an audio buffer and returns the detected note
func (d *FFTDetector) DetectPitch(buffer *audio.AudioBuffer) (*Note, error) {
	if buffer == nil || len(buffer.Samples) == 0 {
		return nil, ErrEmptyBuffer
	}

	// Calculate RMS volume of the buffer
	sumSquares := 0.0
	peakValue := 0.0
	for _, sample := range buffer.Samples {
		sampleVal := float64(sample)
		sumSquares += sampleVal * sampleVal

		// Also track peak value
		absVal := math.Abs(sampleVal)
		if absVal > peakValue {
			peakValue = absVal
		}
	}
	rmsVolume := math.Sqrt(sumSquares / float64(len(buffer.Samples)))

	// Calculate approximate dB level
	dbLevel := -100.0          // Default very low value
	if rmsVolume > 0.0000001 { // Avoid log(0)
		dbLevel = 20 * math.Log10(rmsVolume)
	}

	// Skip everything if the level is too low (likely silence)
	// Use both RMS threshold and dB threshold
	if rmsVolume < d.volumeThreshold || dbLevel < -50.0 {
		return nil, ErrVolumeThreshold
	}

	// If the peak value is too low, also skip (prevents processing very quiet sounds)
	if peakValue < d.volumeThreshold*2 {
		return nil, ErrVolumeThreshold
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

	// Find the fundamental frequency using peak detection
	peakFreq := d.findFundamentalFrequency(spectrum, buffer.SampleRate)

	// If the detected frequency is too low or too high, it's likely noise
	if peakFreq < d.minFrequency || peakFreq > d.maxFrequency {
		return nil, ErrVolumeThreshold
	}

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

// Peak represents a peak in the frequency spectrum
type Peak struct {
	Bin       int
	Magnitude float64
	Frequency float64
}

// findFundamentalFrequency finds the fundamental frequency using improved peak detection
func (d *FFTDetector) findFundamentalFrequency(spectrum []complex128, sampleRate int) float64 {
	// We only need to look at the first half of the spectrum (Nyquist theorem)
	spectrumHalf := spectrum[:len(spectrum)/2]

	// Calculate frequency resolution (Hz per bin)
	binSizeHz := float64(sampleRate) / float64(len(spectrum))

	// Calculate min/max bin numbers based on frequency range
	minBin := int(d.minFrequency / binSizeHz)
	if minBin < 1 {
		minBin = 1 // Avoid DC component
	}

	maxBin := int(d.maxFrequency / binSizeHz)
	if maxBin >= len(spectrumHalf) {
		maxBin = len(spectrumHalf) - 1
	}

	// Find the maximum magnitude for normalization
	maxMagnitude := 0.0
	for i := minBin; i <= maxBin; i++ {
		magnitude := cmplx.Abs(spectrumHalf[i])
		if magnitude > maxMagnitude {
			maxMagnitude = magnitude
		}
	}

	// Don't process further if signal is too weak
	if maxMagnitude < d.noiseFloor {
		return 440.0 // Return A4 as default if no clear signal
	}

	// Find all peaks
	var peaks []Peak
	for i := minBin + 1; i < maxBin; i++ {
		magnitude := cmplx.Abs(spectrumHalf[i])

		// Check if this bin is a peak (higher than adjacent bins)
		if magnitude > cmplx.Abs(spectrumHalf[i-1]) &&
			magnitude > cmplx.Abs(spectrumHalf[i+1]) &&
			magnitude > maxMagnitude*d.peakThreshold {

			// Use quadratic interpolation for more accurate peak location
			// x = 0.5 * (R[k-1] - R[k+1]) / (R[k-1] - 2*R[k] + R[k+1]) + k
			// where R[n] is the magnitude at bin n
			prev := cmplx.Abs(spectrumHalf[i-1])
			current := magnitude
			next := cmplx.Abs(spectrumHalf[i+1])

			// Avoid division by zero
			if prev-2*current+next != 0 {
				delta := 0.5 * (prev - next) / (prev - 2*current + next)

				// Calculate interpolated frequency
				freq := (float64(i) + delta) * binSizeHz

				peaks = append(peaks, Peak{
					Bin:       i,
					Magnitude: magnitude,
					Frequency: freq,
				})
			} else {
				// Just use the bin frequency if we can't interpolate
				peaks = append(peaks, Peak{
					Bin:       i,
					Magnitude: magnitude,
					Frequency: float64(i) * binSizeHz,
				})
			}
		}
	}

	// If no peaks found, return default
	if len(peaks) == 0 {
		return 440.0
	}

	// Sort peaks by magnitude (descending)
	sort.Slice(peaks, func(i, j int) bool {
		return peaks[i].Magnitude > peaks[j].Magnitude
	})

	// The highest peak is our candidate for fundamental frequency
	return peaks[0].Frequency
}
