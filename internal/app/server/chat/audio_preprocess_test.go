package chat

import (
	"math"
	"testing"

	"github.com/spf13/viper"
	"xiaozhi-esp32-server-golang/internal/domain/audio/frontend"
)

func TestAudioPreprocessOffAndShadowPreserveDryPacket(t *testing.T) {
	for _, mode := range []string{"off", "shadow"} {
		t.Run(mode, func(t *testing.T) {
			viper.Reset()
			defer viper.Reset()
			viper.Set("chat.audio_preprocess.mode", mode)
			viper.Set("chat.audio_preprocess.denoise", false)
			p := newAudioPreprocess("test", "session", 16000, 1, 400, true)
			p.Reset()
			defer p.Close()
			pcm := make([]float32, 960)
			for i := range pcm {
				pcm[i] = float32(.1 * math.Sin(float64(i)*.2))
			}
			got, err := p.Process(pcm)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].custom || len(got[0].PCM) != len(pcm) || &got[0].PCM[0] != &pcm[0] {
				t.Fatal("shadow/off altered live PCM")
			}
		})
	}
}
func TestAudioPreprocessEnforceResetAndUnsupportedBypass(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("chat.audio_preprocess.mode", "enforce")
	viper.Set("chat.audio_preprocess.denoise", false)
	p := newAudioPreprocess("test", "session", 16000, 1, 400, true)
	p.Reset()
	defer p.Close()
	frames, err := p.Process(make([]float32, 960))
	if err != nil || len(frames) != 1 || !frames[0].custom {
		t.Fatalf("enforce frame: %v %v", frames, err)
	}
	p.Prepare(frames[0])
	p.Reset()
	// Reset discards the previous epoch's native/FIFO state; 64 samples alone
	// cannot emit a 512-sample hop by borrowing the previous 448-sample tail.
	frames, err = p.Process(make([]float32, 64))
	if err != nil || len(frames) != 0 {
		t.Fatal("old epoch remainder leaked")
	}
	if _, err = p.Process([]float32{float32(math.NaN())}); err == nil {
		t.Fatal("DSP error hidden")
	}
	if _, err = p.Process(make([]float32, 960)); err == nil {
		t.Fatal("dry PCM spliced before epoch boundary")
	}
	p.Reset() // explicit epoch boundary permits dry fallback
	frames, err = p.Process(make([]float32, 960))
	if err != nil || len(frames) != 1 || frames[0].custom {
		t.Fatal("failed DSP did not bypass at fresh epoch")
	}
	for _, eligible := range []bool{false, true} {
		q := newAudioPreprocess("test", "session", 48000, 1, 400, eligible)
		q.Reset()
		frames, err = q.Process(make([]float32, 960))
		q.Close()
		if err != nil || len(frames) != 1 || frames[0].custom {
			t.Fatal("unsupported format must use old pipeline")
		}
	}
}
func TestAudioPreprocessConfirmedShortTurnRetained(t *testing.T) {
	// Acoustic onset classifier tests exercise evidence. Here verify the live
	// adapter retains a confirmed single 32 ms syllable without a duration filter.
	viper.Reset()
	defer viper.Reset()
	viper.Set("chat.audio_preprocess.mode", "enforce")
	viper.Set("chat.audio_preprocess.denoise", false)
	p := newAudioPreprocess("test", "session", 16000, 1, 64, true)
	p.Reset()
	defer p.Close()
	accepted := p.Prepare(preparedAudio{custom: true, Frame: frontend.Frame{PCM: make([]float32, 512), Probability: .95}})
	if !accepted.Voice || !accepted.turn.Started || len(accepted.turn.PCM) != 512 {
		t.Fatal("short syllable lost")
	}
	for i := 0; i < 2; i++ {
		out := p.Prepare(preparedAudio{custom: true, Frame: frontend.Frame{PCM: make([]float32, 512), Probability: .01}})
		if out.turn.Ended {
			t.Fatal("early endpoint")
		}
	}
	out := p.Prepare(preparedAudio{custom: true, Frame: frontend.Frame{PCM: make([]float32, 512), Probability: .01}})
	if !out.turn.Ended || out.turn.VoiceMs != 32 {
		t.Fatal("confirmed short turn not preserved")
	}
	p.EndTurn()
	next := p.Prepare(preparedAudio{custom: true, Frame: frontend.Frame{PCM: make([]float32, 512), Probability: .4}})
	if next.Voice || len(next.turn.PCM) != 0 {
		t.Fatal("onset state leaked after endpoint")
	}
}
