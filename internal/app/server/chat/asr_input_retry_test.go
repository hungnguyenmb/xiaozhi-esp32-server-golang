package chat

import (
	"context"
	"errors"
	"testing"
	"time"
	client "xiaozhi-esp32-server-golang/internal/data/client"
)

func TestASRInputTurnEmptyRetryReopensWithoutTTS(t *testing.T) {
	state := &client.ClientState{ListenMode: "auto", OpusAudioBuffer: make(chan client.OpusInputFrame, 8)}
	state.AudioInput.Configure(true, 300*time.Millisecond)
	old, _ := state.AudioInput.BeginListen("auto", time.Now(), state.OpusAudioBuffer)
	state.AudioInput.Ready(old)
	state.AudioInput.Block() // Endpoint closes input, then ASR yields no text.
	if err := restartASRInputForRetry(state, func() error {
		if !state.AudioInput.Starting() {
			t.Error("input must wait for new ASR stream")
		}
		state.Asr.BeginInputTurn()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if state.AudioInput.Queue(state.OpusAudioBuffer, []byte{1}, time.Now()) != "" {
		t.Fatal("auto input remained blocked after empty ASR")
	}
	frame := <-state.OpusAudioBuffer
	if frame.Discard || !state.AudioInput.WaitFrame(context.Background(), frame.Epoch) || state.AudioInput.Allows(old) {
		t.Fatal("retry admitted wrong epoch or applied playback guard without playback")
	}
	if err := restartASRInputForRetry(state, func() error { return errors.New("provider unavailable") }); err == nil {
		t.Fatal("restart failure hidden")
	}
	if state.AudioInput.Queue(state.OpusAudioBuffer, []byte{2}, time.Now()) != "busy" {
		t.Fatal("failed restart left ASR admission open")
	}
}
