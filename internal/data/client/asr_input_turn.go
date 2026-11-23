package client

import "sync"

type ASRInputStats struct {
	VoiceMs, IdleMs, EndAtMs int64
	Ended                    bool
}

// ASRInputTurn survives VAD.Reset and belongs to one recognition stream, so a
// delayed final result cannot read durations from the next listen turn.
type ASRInputTurn struct {
	mu    sync.Mutex
	stats ASRInputStats
}

func (t *ASRInputTurn) End(voiceMs, idleMs, atMs int64) ASRInputStats {
	if t == nil {
		return ASRInputStats{VoiceMs: voiceMs, IdleMs: idleMs, EndAtMs: atMs, Ended: true}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.stats.Ended {
		t.stats = ASRInputStats{VoiceMs: voiceMs, IdleMs: idleMs, EndAtMs: atMs, Ended: true}
	}
	return t.stats
}
func (t *ASRInputTurn) Snapshot() ASRInputStats {
	if t == nil {
		return ASRInputStats{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stats
}
func (a *Asr) BeginInputTurn()          { a.lock.Lock(); defer a.lock.Unlock(); a.inputTurn = &ASRInputTurn{} }
func (a *Asr) InputTurn() *ASRInputTurn { a.lock.RLock(); defer a.lock.RUnlock(); return a.inputTurn }
