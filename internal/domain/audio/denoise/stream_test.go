package denoise

import (
	"math"
	"os"
	"sync"
	"testing"
)

func requireNative(t *testing.T) {
	t.Helper()
	if os.Getenv("XIAOZHI_TEST_RNNOISE") != "1" {
		t.Skip("set XIAOZHI_TEST_RNNOISE=1 with pinned RNNoise and SpeexDSP")
	}
}
func TestNativePacketBoundariesAndSessionIsolation(t *testing.T) {
	requireNative(t)
	input := make([]float32, 16000)
	for i := range input {
		input[i] = float32(.1 * math.Sin(float64(i)*.13))
	}
	render := func(size int) []float32 {
		s, err := New(16000, 1)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		var out []float32
		for start := 0; start < len(input); start += size {
			end := start + size
			if end > len(input) {
				end = len(input)
			}
			f, err := s.Process(input[start:end])
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, f...)
		}
		if len(out) != len(input) {
			t.Fatalf("size%d lost samples: %d", size, len(out))
		}
		s.Close()
		s.Close()
		if _, err := s.Process(input); err == nil {
			t.Fatal("closed stream accepted audio")
		}
		return out
	}
	want := render(16000)
	for _, size := range []int{1, 40, 320, 640, 960, 1920} {
		got := render(size)
		for i, x := range got {
			if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) || math.Abs(float64(x)) > 1 {
				t.Fatal("invalid output")
			}
			if math.Abs(float64(x-want[i])) > 1e-6 {
				t.Fatalf("packet boundary/session leak size%d sample%d", size, i)
			}
		}
	}
}
func TestNativeConcurrentIndependentStates(t *testing.T) {
	requireNative(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := New(16000, 1)
			if err != nil {
				t.Error(err)
				return
			}
			defer s.Close()
			for j := 0; j < 20; j++ {
				out, err := s.Process(make([]float32, 160))
				if err != nil {
					t.Error(err)
					return
				}
				for _, x := range out {
					if x != 0 {
						t.Error("silence state contaminated")
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}
func TestRejectUnsupportedFormat(t *testing.T) {
	if _, err := New(48000, 1); err == nil {
		t.Fatal("unsupported format accepted")
	}
	if _, err := New(16000, 2); err == nil {
		t.Fatal("stereo accepted")
	}
}
func TestNativeNonFiniteDoesNotPoisonStream(t *testing.T) {
	requireNative(t)
	s, err := New(16000, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Process([]float32{float32(math.NaN())}); err == nil {
		t.Fatal("NaN accepted")
	}
	got, err := s.Process(make([]float32, 160))
	if err != nil || len(got) != 160 {
		t.Fatalf("recovery: %d %v", len(got), err)
	}
}
