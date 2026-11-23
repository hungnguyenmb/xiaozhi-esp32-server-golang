package client

import (
	"context"
	"sync"
	"time"
)

// OpusInputFrame belongs to one listen epoch; queued audio cannot enter a later turn.
type OpusInputFrame struct {
	Data    []byte
	Epoch   uint64
	Discard bool // Decode to prime Opus, but never submit to VAD/ASR.
}

// AudioInputGate controls ASR admission only. Transport recording and commands
// stay outside the gate. Its zero value preserves legacy/realtime behavior.
type AudioInputGate struct {
	mu                           sync.Mutex
	enabled, auto, active, ready bool
	epoch                        uint64
	readyCh                      chan struct{}
	guard                        time.Duration
	settlePending                bool
	settleLatestStart            time.Time
	settleUntil                  time.Time
	naturalStop, wakeUntil       time.Time
}

func (g *AudioInputGate) Configure(enabled bool, guard time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = enabled
	g.auto = true
	g.guard = guard
}
func (g *AudioInputGate) notify() {
	if g.readyCh != nil {
		close(g.readyCh)
		g.readyCh = nil
	}
}
func (g *AudioInputGate) BeginListen(mode string, now time.Time, queue chan OpusInputFrame) (uint64, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.notify()
	g.epoch++
	g.auto = mode == "auto"
	g.active = true
	g.ready = false
	g.readyCh = make(chan struct{})
	guarded := g.enabled && g.auto && !g.naturalStop.IsZero() && now.Sub(g.naturalStop) >= 0 && now.Sub(g.naturalStop) < 2*time.Second && now.After(g.wakeUntil) && g.guard > 0
	g.settleLatestStart = now.Add(time.Second)
	g.settlePending = guarded
	g.settleUntil = time.Time{}
	g.naturalStop = time.Time{}
	g.wakeUntil = time.Time{}
	// Producers also hold mu, so no old frame can be queued after this drain.
	for {
		select {
		case _, ok := <-queue:
			if !ok {
				return g.epoch, guarded
			}
		default:
			return g.epoch, guarded
		}
	}
}
func (g *AudioInputGate) Ready(epoch uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if epoch == g.epoch {
		g.ready = true
		g.notify()
	}
}
func (g *AudioInputGate) Block() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.enabled && g.auto {
		g.active = false
		g.epoch++
		g.notify()
	}
}
func (g *AudioInputGate) PlaybackStopped(now time.Time, natural bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if natural {
		g.naturalStop = now
	}
}
func (g *AudioInputGate) Wake(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.wakeUntil = now.Add(2 * time.Second)
	g.naturalStop = time.Time{}
	g.settlePending = false
	g.settleUntil = time.Time{}
}

// Queue returns a policy reason for intentionally discarded frames. "full" is
// the only failure that the caller should report as an overflowing buffer.
func (g *AudioInputGate) Queue(queue chan OpusInputFrame, data []byte, now time.Time) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	reason := ""
	if g.enabled && g.auto {
		if !g.active {
			return "busy"
		}
		if g.settlePending {
			g.settlePending = false
			if now.Before(g.settleLatestStart) {
				g.settleUntil = now.Add(g.guard)
				reason = "settling_start"
			}
		}
		if reason == "" && now.Before(g.settleUntil) {
			reason = "settling"
		}
	}
	select {
	case queue <- OpusInputFrame{Data: data, Epoch: g.epoch, Discard: reason != ""}:
		return reason
	default:
		return "full"
	}
}

func (g *AudioInputGate) WaitFrame(ctx context.Context, epoch uint64) bool {
	g.mu.Lock()
	if !g.enabled || !g.auto {
		allowed := !g.enabled || epoch == g.epoch
		g.mu.Unlock()
		return allowed
	}
	if epoch != g.epoch || !g.active {
		g.mu.Unlock()
		return false
	}
	ch := g.readyCh
	ready := g.ready
	g.mu.Unlock()
	if !ready {
		select {
		case <-ctx.Done():
			return false
		case <-ch:
		}
	}
	return g.Allows(epoch)
}
func (g *AudioInputGate) Allows(epoch uint64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return !g.enabled || (epoch == g.epoch && (!g.auto || (g.active && g.ready)))
}

func (g *AudioInputGate) Epoch() uint64 { g.mu.Lock(); defer g.mu.Unlock(); return g.epoch }

// Forward linearizes the final ASR write against Block/BeginListen. The callback
// must be non-blocking and must not call back into this gate.
func (g *AudioInputGate) Forward(epoch uint64, send func()) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.enabled && (epoch != g.epoch || (g.auto && (!g.active || !g.ready))) {
		return false
	}
	send()
	return true
}

// Starting accepts buffering while the old VoiceStop flag is still being reset.
func (g *AudioInputGate) Starting() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.enabled && g.auto && g.active && !g.ready
}
