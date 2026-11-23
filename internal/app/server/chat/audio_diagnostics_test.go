package chat

import (
	"math"
	"testing"
	"time"
)

type diagnosticScore struct{}

func (diagnosticScore) DiagnosticProbability() (float32, bool) { return .8, true }

func TestAudioDiagnosticsWindow(t *testing.T) {
	d := &audioDiagnostics{enabled: true, interval: time.Hour, since: time.Now()}
	d.observe([]float32{.5, -.5}, true, true, false, diagnosticScore{}, 20, "ten_vad")
	d.observe([]float32{0, 0}, false, false, true, nil, 20, "ten_vad")
	if math.Abs(d.rms()-math.Sqrt(.125)) > 1e-8 || d.peak != .5 || d.frames != 2 || d.voiceFrames != 1 || d.powerSkips != 1 || d.vadCalls != 1 || d.scores != 1 || d.audioMs != 40 {
		t.Fatalf("unexpected diagnostics: %+v", d)
	}
	d.flush("test")
	if d.frames != 0 || d.samples != 0 || d.scores != 0 || !d.enabled || d.interval != time.Hour {
		t.Fatalf("window not reset: %+v", d)
	}
}

func TestAudioDiagnosticsDisabledAndUnsupportedScore(t *testing.T) {
	d := &audioDiagnostics{}
	d.observe([]float32{1}, true, true, false, diagnosticScore{}, 20, "ten_vad")
	if d.frames != 0 {
		t.Fatal("disabled diagnostics collected samples")
	}
	d.enabled = true
	d.since = time.Now()
	d.interval = time.Hour
	d.observe([]float32{0}, false, true, false, struct{}{}, 20, "other")
	if d.scores != 0 || d.vadCalls != 1 {
		t.Fatal("unsupported score must remain unavailable")
	}
}
