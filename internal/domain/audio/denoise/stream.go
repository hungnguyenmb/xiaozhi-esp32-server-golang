// Package denoise provides a per-epoch, CPU RNNoise stream. It is optional at
// runtime: unavailable libraries are an explicit initialization error, not mute.
package denoise

import (
	"fmt"
	"math"
)

// Stream is single-owner. Only 16 kHz mono is enabled in this first release.
// Native RNNoise runs at 48 kHz in 10 ms blocks via stateful Speex resamplers.
type Stream struct {
	native  *nativeStream
	pending []float32
	closed  bool
}

func New(sampleRate, channels int) (*Stream, error) {
	if sampleRate != 16000 || channels != 1 {
		return nil, fmt.Errorf("RNNoise requires 16000 Hz mono input, got %d/%d", sampleRate, channels)
	}
	n, err := newNative()
	if err != nil {
		return nil, err
	}
	return &Stream{native: n}, nil
}

// Process preserves all samples across arbitrary packet boundaries. At most
// 159 samples await the next call. Output includes the native algorithmic delay;
// it is never spliced with dry PCM. Trailing silence carries the delayed tail.
func (s *Stream) Process(pcm []float32) ([]float32, error) {
	if s.closed {
		return nil, fmt.Errorf("denoiser is closed")
	}
	for _, x := range pcm {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return nil, fmt.Errorf("non-finite PCM")
		}
	}
	s.pending = append(s.pending, pcm...)
	out := make([]float32, 0, len(s.pending)/160*160)
	for len(s.pending) >= 160 {
		frame, err := s.native.process(s.pending[:160])
		if err != nil {
			return nil, err
		}
		out = append(out, frame...)
		s.pending = s.pending[160:]
	}
	// Do not retain a large packet's backing array between calls.
	s.pending = append([]float32(nil), s.pending...)
	return out, nil
}

func (s *Stream) Close() {
	if !s.closed {
		s.native.close()
		s.pending = nil
		s.closed = true
	}
}

// LatencySamples reports fixed native/resampler delay, excluding FIFO and network.
func (s *Stream) LatencySamples() int {
	if s.closed {
		return 0
	}
	return s.native.latencySamples()
}
