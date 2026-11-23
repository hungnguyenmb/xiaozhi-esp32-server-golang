//go:build cgo

package ten_vad

import (
	"math"
	"testing"
)

func TestResetRestoresFreshNativeState(t *testing.T) {
	config := map[string]interface{}{"hop_size": 256, "threshold": 0.5, "threshold_low": 0.2}
	vad, err := NewTenVAD(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vad.Close() })
	native := GetInstance()
	frame := make([]int16, 256)
	for i := range frame {
		frame[i] = int16(16000 * math.Sin(float64(i)*0.17))
	}
	for i := 0; i < 20; i++ {
		if _, _, err := native.ProcessAudio(vad.handle, frame); err != nil {
			t.Fatal(err)
		}
	}
	vad.detector.Evaluate(1)
	if err := vad.Reset(); err != nil {
		t.Fatal(err)
	}
	if vad.detector.Active() {
		t.Fatal("hysteresis speech state survived reset")
	}
	fresh, err := NewTenVAD(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fresh.Close() })
	for i := 0; i < 12; i++ {
		got, _, err := native.ProcessAudio(vad.handle, frame)
		if err != nil {
			t.Fatal(err)
		}
		want, _, err := native.ProcessAudio(fresh.handle, frame)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(float64(got-want)) > 1e-6 {
			t.Fatalf("native state leaked across reset at frame %d: got %v want %v", i, got, want)
		}
	}
}

func TestResetDoesNotReopenClosedVAD(t *testing.T) {
	vad, err := NewTenVAD(map[string]interface{}{"hop_size": 256})
	if err != nil {
		t.Fatal(err)
	}
	if err := vad.Close(); err != nil {
		t.Fatal(err)
	}
	if vad.Reset() == nil || vad.IsValid() {
		t.Fatal("reset must not resurrect a closed instance")
	}
}
