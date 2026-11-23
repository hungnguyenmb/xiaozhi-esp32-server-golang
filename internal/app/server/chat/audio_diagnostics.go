package chat

import (
	"math"
	"time"

	"github.com/spf13/viper"
	log "xiaozhi-esp32-server-golang/logger"
)

type audioDiagnostics struct {
	enabled                                                    bool
	device, session, provider                                  string
	interval                                                   time.Duration
	since                                                      time.Time
	frames, samples, voiceFrames, vadCalls, powerSkips, scores int
	audioMs                                                    int
	squares, probabilitySum                                    float64
	peak, probabilityMin, probabilityMax                       float32
}

func newAudioDiagnostics(device, session string) *audioDiagnostics {
	interval := viper.GetInt("chat.audio_diagnostics.interval_ms")
	if interval <= 0 {
		interval = 1000
	}
	if interval < 500 {
		interval = 500
	}
	if interval > 10000 {
		interval = 10000
	}
	return &audioDiagnostics{enabled: viper.GetBool("chat.audio_diagnostics.enabled"), device: device, session: session, interval: time.Duration(interval) * time.Millisecond, since: time.Now()}
}

// observe is owned exclusively by the audio processing goroutine. Scores are
// sampled once per VAD call, not once per internal inference window.
func (d *audioDiagnostics) observe(pcm []float32, voice, called, powerSkip bool, provider any, frameMs int, name string) {
	if !d.enabled {
		return
	}
	d.provider = name
	d.frames++
	d.audioMs += frameMs
	if voice {
		d.voiceFrames++
	}
	if powerSkip {
		d.powerSkips++
	}
	if called {
		d.vadCalls++
		if source, ok := provider.(interface{ DiagnosticProbability() (float32, bool) }); ok {
			if score, valid := source.DiagnosticProbability(); valid {
				if d.scores == 0 || score < d.probabilityMin {
					d.probabilityMin = score
				}
				if d.scores == 0 || score > d.probabilityMax {
					d.probabilityMax = score
				}
				d.scores++
				d.probabilitySum += float64(score)
			}
		}
	}
	for _, sample := range pcm {
		value := float64(sample)
		d.squares += value * value
		abs := float32(math.Abs(value))
		if abs > d.peak {
			d.peak = abs
		}
	}
	d.samples += len(pcm)
	if time.Since(d.since) >= d.interval {
		d.flush("interval")
	}
}

func (d *audioDiagnostics) rms() float64 {
	if d.samples == 0 {
		return 0
	}
	return math.Sqrt(d.squares / float64(d.samples))
}

func (d *audioDiagnostics) flush(reason string) {
	if !d.enabled || d.frames == 0 {
		return
	}
	scoreMean := float64(0)
	if d.scores > 0 {
		scoreMean = d.probabilitySum / float64(d.scores)
	}
	log.Infof("AUDIO_DIAG window device=%s session=%s provider=%s reason=%s wall_ms=%d audio_ms=%d frames=%d voice_frames=%d vad_calls=%d power_skips=%d rms=%.6f peak=%.6f score_samples=%d score_min=%.4f score_mean=%.4f score_max=%.4f", d.device, d.session, d.provider, reason, time.Since(d.since).Milliseconds(), d.audioMs, d.frames, d.voiceFrames, d.vadCalls, d.powerSkips, d.rms(), d.peak, d.scores, d.probabilityMin, scoreMean, d.probabilityMax)
	*d = audioDiagnostics{enabled: d.enabled, device: d.device, session: d.session, provider: d.provider, interval: d.interval, since: time.Now()}
}

func (d *audioDiagnostics) event(reason string, voiceMs, idleMs int64) {
	if !d.enabled {
		return
	}
	d.flush(reason)
	log.Infof("AUDIO_DIAG endpoint device=%s session=%s reason=%s voice_ms=%d idle_ms=%d", d.device, d.session, reason, voiceMs, idleMs)
}

func logASRDiagnostic(device, session, stage string, chars int, voiceMs, asrMs, idleMs int64) {
	if !viper.GetBool("chat.audio_diagnostics.enabled") {
		return
	}
	log.Infof("AUDIO_DIAG asr device=%s session=%s stage=%s text_chars=%d voice_ms=%d asr_ms=%d idle_ms=%d", device, session, stage, chars, voiceMs, asrMs, idleMs)
}
