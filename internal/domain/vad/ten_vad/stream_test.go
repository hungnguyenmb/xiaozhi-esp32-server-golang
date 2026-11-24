package ten_vad

import (
	"math"
	"testing"
)

func TestPacketBoundariesKeepNativeSamples(t *testing.T) {
	config := map[string]interface{}{"hop_size": 512, "threshold": 0.4}
	reference, err := NewTenVAD(config)
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	input := make([]float32, 16000)
	for i := range input {
		input[i] = float32(.2 * math.Sin(float64(i)*.11))
	}
	if _, err = reference.IsVADExt(input, 16000, len(input)); err != nil {
		t.Fatal(err)
	}
	want, _ := reference.DiagnosticProbability()
	for _, size := range []int{1, 320, 640, 960, 1920} {
		vad, err := NewTenVAD(config)
		if err != nil {
			t.Fatal(err)
		}
		var last float32
		for start := 0; start < len(input); start += size {
			end := start + size
			if end > len(input) {
				end = len(input)
			}
			if _, err = vad.IsVADExt(input[start:end], 16000, end-start); err != nil {
				t.Fatal(err)
			}
			if p, ok := vad.DiagnosticProbability(); ok {
				last = p
			}
		}
		if math.Abs(float64(last-want)) > 1e-6 {
			t.Fatalf("size %d native state differs: %v != %v", size, last, want)
		}
		if len(vad.pending) != len(input)%512 {
			t.Fatalf("size %d remainder lost", size)
		}
		if err = vad.Reset(); err != nil {
			t.Fatal(err)
		}
		if len(vad.pending) != 0 {
			t.Fatal("reset retained another epoch's samples")
		}
		vad.Close()
	}
}

func TestPCMInt16Saturation(t *testing.T) {
	for _, tc := range []struct {
		in   float32
		want int16
	}{{1, 32767}, {2, 32767}, {-1, -32768}, {-2, -32768}, {.5, 16384}, {float32(math.NaN()), 0}} {
		if got := pcmInt16(tc.in); got != tc.want {
			t.Fatalf("%v: %d != %d", tc.in, got, tc.want)
		}
	}
}
