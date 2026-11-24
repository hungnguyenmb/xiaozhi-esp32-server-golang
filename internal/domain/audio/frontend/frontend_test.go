package frontend

import (
	"math"
	"testing"
)

func TestOnsetKeepsStrongShortSyllableAndWeakGap(t *testing.T) {
	c := DefaultConfig()
	for _, scores := range [][]float32{{.9}, {.6, .6}, {.6, .1, .6}} {
		var d onsetDetector
		var accepted bool
		for _, p := range scores {
			v, _ := d.observe(p, c)
			accepted = accepted || v
		}
		if !accepted {
			t.Fatalf("short utterance lost: %v", scores)
		}
		if v, _ := d.observe(.4, c); !v {
			t.Fatal("weak tail cut after acceptance")
		}
		if v, _ := d.observe(.1, c); v {
			t.Fatal("silence labeled voice")
		}
	}
	var d onsetDetector
	for _, p := range []float32{.1, .6, .1, .1, .1} {
		if v, _ := d.observe(p, c); v {
			t.Fatal("isolated medium impulse accepted")
		}
	}
}
func TestTurnKeepsOnsetWeakInteriorAndEndpointExactlyOnce(t *testing.T) {
	turn := NewTurn(64)
	input := []Frame{
		{PCM: []float32{1, 2}, Voice: false},
		{PCM: []float32{3, 4}, Voice: true},
		{PCM: []float32{5, 6}, Voice: false},
		{PCM: []float32{7, 8}, Voice: true},
	}
	var got []float32
	starts := 0
	for _, f := range input {
		out := turn.Push(f)
		if out.Started {
			starts++
		}
		got = append(got, out.PCM...)
	}
	if starts != 1 || len(got) != 8 {
		t.Fatalf("start/pre-roll duplicate: %d %v", starts, got)
	}
	for i, x := range got {
		if x != float32(i+1) {
			t.Fatal("onset/interior samples changed")
		}
	}
	for i := 0; i < 2; i++ {
		if turn.Push(Frame{PCM: make([]float32, Hop)}).Ended {
			t.Fatal("endpoint before strict silence threshold")
		}
	}
	if !turn.Push(Frame{PCM: make([]float32, Hop)}).Ended {
		t.Fatal("missing endpoint")
	}
	turn.End()
	if out := turn.Push(Frame{PCM: []float32{9}, Voice: true}); !out.Started || len(out.PCM) != 1 || out.PCM[0] != 9 {
		t.Fatal("old turn leaked")
	}
}
func TestPreRollBoundedDuringLongBackground(t *testing.T) {
	turn := NewTurn(400)
	for i := 0; i < 10000; i++ {
		turn.Push(Frame{PCM: make([]float32, Hop)})
	}
	if len(turn.pending) > 3200 {
		t.Fatal("unbounded pre-roll")
	}
	out := turn.Push(Frame{PCM: make([]float32, Hop), Voice: true})
	if len(out.PCM) != 3200+Hop {
		t.Fatal("pre-roll trimmed current frame")
	}
}
func TestFrontendHopStreamIndependentOfPacketSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Denoise = false
	input := make([]float32, 16000)
	for i := range input {
		input[i] = float32(.1 * math.Sin(float64(i)*.12))
	}
	render := func(size int) []Frame {
		p, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()
		var out []Frame
		for start := 0; start < len(input); start += size {
			end := start + size
			if end > len(input) {
				end = len(input)
			}
			f, err := p.Process(input[start:end])
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, f...)
		}
		return out
	}
	want := render(len(input))
	for _, size := range []int{40, 320, 640, 960} {
		got := render(size)
		if len(got) != len(want) {
			t.Fatal("hop count differs")
		}
		for i, f := range got {
			if f.Voice != want[i].Voice || f.Probability != want[i].Probability {
				t.Fatalf("size%d decision%d differs", size, i)
			}
		}
	}
}
func TestInvalidThresholds(t *testing.T) {
	for _, c := range []Config{{Start: .3, Continue: .4, Strong: .8}, {Start: .4, Continue: .2, Strong: .1}, {Start: float32(math.NaN())}} {
		if c.Validate() == nil {
			t.Fatal("invalid config accepted")
		}
	}
}
