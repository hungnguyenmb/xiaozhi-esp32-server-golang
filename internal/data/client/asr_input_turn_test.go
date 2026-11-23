package client

import (
	"context"
	"testing"
)

func TestASRInputTurnSurvivesVoiceSilenceReset(t *testing.T) {
	state := &ClientState{Ctx: context.Background()}
	state.Asr.BeginInputTurn()
	old := state.Asr.InputTurn()
	state.Vad.AddVoiceDuration(420)
	state.Vad.AddIdleDuration(540)
	state.OnVoiceSilence()
	if state.Vad.GetVoiceDurationInSession() != 0 {
		t.Fatal("expected existing VAD reset")
	}
	stats := old.Snapshot()
	if !stats.Ended || stats.VoiceMs != 420 || stats.IdleMs != 540 || stats.EndAtMs <= 0 {
		t.Fatalf("snapshot lost: %+v", stats)
	}
	state.OnVoiceSilence()
	if old.Snapshot() != stats {
		t.Fatal("second silence overwrote snapshot with zeros")
	}
	state.Asr.BeginInputTurn()
	if state.Asr.InputTurn() == old || state.Asr.InputTurn().Snapshot().Ended {
		t.Fatal("new turn reused stats")
	}
	if old.Snapshot() != stats {
		t.Fatal("new turn corrupted late result snapshot")
	}
}
