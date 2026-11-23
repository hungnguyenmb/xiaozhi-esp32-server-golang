package chat

import (
	"testing"
	client "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/asr"
	"xiaozhi-esp32-server-golang/internal/pool"
)

func TestASRInputTurnStaleLoopDoesNotReleaseNewResource(t *testing.T) {
	state := &client.ClientState{}
	state.Asr.BeginInputTurn()
	old := state.Asr.InputTurn()
	state.Asr.BeginInputTurn()
	current := &pool.ResourceWrapper[asr.AsrProvider]{}
	manager := NewASRManager(state, nil)
	manager.asrResource = current
	manager.releaseInputTurnResource(old)
	if manager.asrResource != current {
		t.Fatal("stale loop released current resource")
	}
}
