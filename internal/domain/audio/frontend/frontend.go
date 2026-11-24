// Package frontend is the optional streaming speech front end shared by live
// input and offline replay. It owns no network connections or global state.
package frontend

import (
	"fmt"
	"math"
	"xiaozhi-esp32-server-golang/internal/domain/audio/denoise"
	"xiaozhi-esp32-server-golang/internal/domain/vad/ten_vad"
)

const SampleRate = 16000
const Hop = 512 // TEN native window: 32 ms, independent of Opus packet duration.

type Config struct {
	Denoise                 bool
	Start, Continue, Strong float32
}

func DefaultConfig() Config { return Config{Denoise: true, Start: .55, Continue: .35, Strong: .85} }
func (c Config) Validate() error {
	if math.IsNaN(float64(c.Start)) || math.IsNaN(float64(c.Continue)) || math.IsNaN(float64(c.Strong)) || c.Continue < 0 || c.Start <= 0 || c.Continue > c.Start || c.Strong < c.Start || c.Strong > 1 {
		return fmt.Errorf("invalid frontend thresholds: need 0 <= continue <= start <= strong <= 1")
	}
	return nil
}

type Frame struct {
	PCM         []float32
	Voice       bool
	Probability float32
	Event       string // candidate, accepted, rejected; empty for steady state
}
type Processor struct {
	cfg     Config
	noise   *denoise.Stream
	vad     *ten_vad.TenVAD
	pending []float32
	onset   onsetDetector
	closed  bool
}

func New(cfg Config) (*Processor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	p := &Processor{cfg: cfg}
	var err error
	p.vad, err = ten_vad.NewTenVAD(map[string]interface{}{"hop_size": Hop, "threshold": cfg.Start, "threshold_low": cfg.Continue})
	if err != nil {
		return nil, err
	}
	if cfg.Denoise {
		p.noise, err = denoise.New(SampleRate, 1)
		if err != nil {
			p.Close()
			return nil, err
		}
	}
	return p, nil
}
func (p *Processor) Process(pcm []float32) ([]Frame, error) {
	if p.closed {
		return nil, fmt.Errorf("frontend closed")
	}
	for _, x := range pcm {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return nil, fmt.Errorf("non-finite input")
		}
	}
	var err error
	if p.noise != nil {
		pcm, err = p.noise.Process(pcm)
		if err != nil {
			return nil, err
		}
	}
	p.pending = append(p.pending, pcm...)
	var frames []Frame
	for len(p.pending) >= Hop {
		samples := append([]float32(nil), p.pending[:Hop]...)
		p.pending = p.pending[Hop:]
		if _, err = p.vad.IsVADExt(samples, SampleRate, Hop); err != nil {
			return nil, err
		}
		probability, valid := p.vad.DiagnosticProbability()
		if !valid {
			return nil, fmt.Errorf("TEN returned no score for full hop")
		}
		frames = append(frames, Frame{PCM: samples, Probability: probability})
	}
	p.pending = append([]float32(nil), p.pending...)
	return frames, nil
}

// EndTurn resets onset evidence after an endpoint; DSP/native temporal history
// continues until the next epoch. Never call it for each unvoiced frame.
func (p *Processor) EndTurn() { p.onset = onsetDetector{} }
func (p *Processor) Close() {
	if p.closed {
		return
	}
	p.closed = true
	if p.noise != nil {
		p.noise.Close()
	}
	if p.vad != nil {
		_ = p.vad.Close()
	}
	p.pending = nil
}

type onsetDetector struct {
	active   bool
	evidence uint8
}

func (d *onsetDetector) observe(probability float32, c Config) (bool, string) {
	if d.active {
		return probability >= c.Continue, ""
	}
	// Two start hits in three native windows (max 96 ms), or one strong window.
	// A brief weak gap does not erase a real short syllable. Pre-roll is retained
	// by the caller until acceptance, so confirmation never trims the onset.
	previous := d.evidence
	d.evidence = (d.evidence << 1) & 7
	if probability >= c.Start {
		d.evidence |= 1
	}
	hits := 0
	for bits := d.evidence; bits != 0; bits >>= 1 {
		hits += int(bits & 1)
	}
	if probability >= c.Strong || hits >= 2 {
		d.active = true
		return true, "accepted"
	}
	if d.evidence != 0 {
		return false, "candidate"
	}
	if previous != 0 {
		return false, "rejected"
	}
	return false, ""
}

// Turn assembles accepted audio without deleting weak hops within a sentence.
// It is also used by live input, making replay share pre-roll and endpoint rules.
type Turn struct {
	silenceSamples            int
	pending                   []float32
	active                    bool
	idleSamples, voiceSamples int
}
type TurnOutput struct {
	PCM             []float32
	Started, Ended  bool
	VoiceMs, IdleMs int64
}

func NewTurn(silenceMs int) *Turn {
	if silenceMs < 1 {
		silenceMs = 500
	}
	return &Turn{silenceSamples: silenceMs * SampleRate / 1000}
}
func (t *Turn) Push(f Frame) TurnOutput {
	var out TurnOutput
	if !t.active {
		t.pending = append(t.pending, f.PCM...)
		if !f.Voice {
			limit := SampleRate * 200 / 1000
			if len(t.pending) > limit {
				t.pending = append([]float32(nil), t.pending[len(t.pending)-limit:]...)
			}
			return out
		}
		out.Started = true
		t.active = true
		out.PCM = append([]float32(nil), t.pending...)
		t.pending = nil
	} else {
		out.PCM = f.PCM
	}
	if f.Voice {
		t.voiceSamples += len(f.PCM)
		t.idleSamples = 0
	} else {
		t.idleSamples += len(f.PCM)
	}
	out.VoiceMs = int64(t.voiceSamples) * 1000 / SampleRate
	out.IdleMs = int64(t.idleSamples) * 1000 / SampleRate
	out.Ended = t.idleSamples > t.silenceSamples
	// Caller confirms endpoint (e.g. provider may wait for first ASR text).
	return out
}

func (t *Turn) End() { t.active = false; t.pending = nil; t.voiceSamples = 0; t.idleSamples = 0 }

// Classify is separate from native inference so callers can commit an endpoint
// between two hops produced by the same packet before classifying the next hop.
func (p *Processor) Classify(f Frame) Frame {
	f.Voice, f.Event = p.onset.observe(f.Probability, p.cfg)
	return f
}

// NativeDelayMs excludes hop/packet buffering and onset confirmation.
func (p *Processor) NativeDelayMs() float64 {
	if p.noise == nil {
		return 0
	}
	return float64(p.noise.LatencySamples()) * 1000 / SampleRate
}
