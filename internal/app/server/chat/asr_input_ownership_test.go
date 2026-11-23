package chat

import (
	"context"
	"testing"
	"time"
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

func TestASRInputTurnCancelledResultDoesNotQueueLLM(t *testing.T) {
	session, state := newAbortTestSession("auto")
	state.Asr.BeginInputTurn()
	turn := state.Asr.InputTurn()
	manager := NewASRManager(state, nil)
	manager.session = session
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.addAsrResultToQueue(ctx, turn, "old", nil); err != nil {
		t.Fatal(err)
	}
	// A superseded turn must also be rejected even if its caller forgot to cancel.
	state.Asr.BeginInputTurn()
	if err := manager.addAsrResultToQueue(context.Background(), turn, "old", nil); err != nil {
		t.Fatal(err)
	}
	timeout, done := context.WithTimeout(context.Background(), time.Millisecond)
	defer done()
	if item, err := session.chatTextQueue.Pop(timeout, 0); err == nil {
		t.Fatalf("stale result reached LLM queue: %+v", item)
	}
}
