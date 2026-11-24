package chat

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/viper"
	"xiaozhi-esp32-server-golang/internal/domain/audio/frontend"
	log "xiaozhi-esp32-server-golang/logger"
)

type preparedAudio struct {
	frontend.Frame
	raw    []float32
	custom bool
	turn   frontend.TurnOutput
}

func (f preparedAudio) DiagnosticProbability() (float32, bool) { return f.Probability, f.custom }

type audioPreprocess struct {
	mode                         string
	epoch                        uint64
	pendingRaw                   []float32
	needsEpoch                   bool
	cfg                          frontend.Config
	processor                    *frontend.Processor
	turn                         *frontend.Turn
	device, session              string
	rate, channels, silence      int
	eligible, failed             bool
	frames, accepted, candidates int
	rawSquares, cleanSquares     float64
	rawSamples, cleanSamples     int
	elapsed                      time.Duration
	since                        time.Time
}

func newAudioPreprocess(device, session string, rate, channels, silence int, eligible bool) *audioPreprocess {
	p := &audioPreprocess{mode: viper.GetString("chat.audio_preprocess.mode"), cfg: frontend.DefaultConfig(), device: device, session: session, rate: rate, channels: channels, silence: silence, eligible: eligible, since: time.Now()}
	if p.mode == "" {
		p.mode = "off"
	}
	for name, dst := range map[string]*float32{"start": &p.cfg.Start, "continue": &p.cfg.Continue, "strong": &p.cfg.Strong} {
		key := "chat.audio_preprocess." + name
		if viper.IsSet(key) {
			*dst = float32(viper.GetFloat64(key))
		}
	}
	if viper.IsSet("chat.audio_preprocess.denoise") {
		p.cfg.Denoise = viper.GetBool("chat.audio_preprocess.denoise")
	}
	if p.mode != "off" && p.mode != "shadow" && p.mode != "enforce" {
		p.disable(fmt.Errorf("invalid mode %q", p.mode))
	}
	return p
}

// Reset is called at an epoch boundary, never in the middle of delayed PCM.
func (p *audioPreprocess) Reset() {
	p.Close()
	p.needsEpoch = false
	if p.failed {
		return
	}
	if p.mode == "off" {
		return
	}
	if !p.eligible || p.rate != frontend.SampleRate || p.channels != 1 {
		p.disable(fmt.Errorf("requires server-endpoint TEN-VAD at 16 kHz mono"))
		return
	}
	var err error
	p.processor, err = frontend.New(p.cfg)
	if err != nil {
		p.disable(err)
		return
	}
	p.turn = frontend.NewTurn(p.silence)
	log.Infof("AUDIO_PREPROCESS setup device=%s session=%s mode=%s epoch=%d denoise=%t start=%.3f continue=%.3f strong=%.3f hop_ms=32 pre_roll_ms=200 native_delay_ms=%.2f", p.device, p.session, p.mode, p.epoch, p.cfg.Denoise, p.cfg.Start, p.cfg.Continue, p.cfg.Strong, p.processor.NativeDelayMs())
}
func (p *audioPreprocess) disable(err error) {
	p.failed = true
	log.Warnf("AUDIO_PREPROCESS bypass device=%s session=%s mode=%s error=%v", p.device, p.session, p.mode, err)
}
func (p *audioPreprocess) Close() {
	if p.processor != nil {
		p.processor.Close()
		p.processor = nil
	}
	p.turn = nil
	p.pendingRaw = nil
}
func (p *audioPreprocess) EndTurn() {
	if p.processor != nil {
		p.processor.EndTurn()
	}
	if p.turn != nil {
		p.turn.End()
	}
}
func (p *audioPreprocess) Process(pcm []float32) ([]preparedAudio, error) {
	dry := []preparedAudio{{Frame: frontend.Frame{PCM: pcm}}}
	if p.needsEpoch {
		return nil, fmt.Errorf("DSP failure requires a new input epoch")
	}
	if p.processor == nil {
		return dry, nil
	}
	start := time.Now()
	frames, err := p.processor.Process(pcm)
	p.elapsed += time.Since(start)
	if err != nil {
		// Do not splice dry samples into an active delayed/denoised turn. The live
		// caller closes/retries the ASR turn; bypass begins only in that new epoch.
		p.Close()
		p.disable(err)
		p.needsEpoch = p.mode == "enforce"
		if p.mode == "shadow" {
			return dry, nil
		}
		return nil, err
	}
	for _, x := range pcm {
		p.rawSquares += float64(x) * float64(x)
	}
	p.rawSamples += len(pcm)
	p.pendingRaw = append(p.pendingRaw, pcm...)
	result := make([]preparedAudio, 0, len(frames))
	for _, f := range frames {
		raw := append([]float32(nil), p.pendingRaw[:len(f.PCM)]...)
		p.pendingRaw = p.pendingRaw[len(f.PCM):]
		prepared := preparedAudio{Frame: f, custom: true, raw: raw}
		if p.mode == "shadow" {
			prepared = p.Prepare(prepared)
			if prepared.turn.Ended {
				p.EndTurn()
			}
		}
		result = append(result, prepared)
	}

	p.pendingRaw = append([]float32(nil), p.pendingRaw...)
	if time.Since(p.since) >= time.Second {
		rms := func(sum float64, n int) float64 {
			if n == 0 {
				return 0
			}
			return math.Sqrt(sum / float64(n))
		}
		log.Infof("AUDIO_PREPROCESS interval device=%s session=%s mode=%s frames=%d accepted=%d candidates=%d raw_rms=%.6f clean_rms=%.6f processing_ms=%.3f", p.device, p.session, p.mode, p.frames, p.accepted, p.candidates, rms(p.rawSquares, p.rawSamples), rms(p.cleanSquares, p.cleanSamples), float64(p.elapsed)/float64(time.Millisecond))
		p.frames = 0
		p.accepted = 0
		p.candidates = 0
		p.rawSquares = 0
		p.cleanSquares = 0
		p.rawSamples = 0
		p.cleanSamples = 0
		p.elapsed = 0
		p.since = time.Now()
	}
	if p.mode == "shadow" {
		return dry, nil
	}
	return result, nil
}

// Prepare commits onset/turn state in the audio consumer, after epoch validation.
// Thus an endpoint between two hops in one Opus packet affects the next hop.
func (p *audioPreprocess) Prepare(f preparedAudio) preparedAudio {
	if !f.custom {
		return f
	}
	f.Frame = p.processor.Classify(f.Frame)
	f.turn = p.turn.Push(f.Frame)
	p.frames++
	if f.Event == "accepted" {
		p.accepted++
		log.Infof("AUDIO_PREPROCESS accepted device=%s session=%s epoch=%d score=%.4f", p.device, p.session, p.epoch, f.Probability)
	}
	if f.Event == "candidate" {
		p.candidates++
	}
	for _, x := range f.PCM {
		p.cleanSquares += float64(x) * float64(x)
	}
	p.cleanSamples += len(f.PCM)
	return f
}
