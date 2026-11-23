package client

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAudioInputGateAutoPlaybackCycle(t *testing.T) {
	now := time.Unix(100, 0)
	var gate AudioInputGate
	gate.Configure(true, 300*time.Millisecond)
	queue := make(chan OpusInputFrame, 10)
	first, guarded := gate.BeginListen("auto", now, queue)
	if guarded {
		t.Fatal("first listen must not settle")
	}
	gate.Ready(first)
	if reason := gate.Queue(queue, []byte{1}, now); reason != "" {
		t.Fatal(reason)
	}
	old := <-queue
	called := false
	if !gate.Forward(old.Epoch, func() { called = true }) || !called {
		t.Fatal("first command not forwarded")
	}
	gate.Block()
	if gate.Queue(queue, []byte{2}, now) != "busy" || gate.Forward(old.Epoch, func() { t.Error("stale frame reached ASR") }) {
		t.Fatal("processing gate failed")
	}
	gate.PlaybackStopped(now.Add(time.Second), true)
	epoch, guarded := gate.BeginListen("auto", now.Add(1080*time.Millisecond), queue)
	if !guarded {
		t.Fatal("natural auto resume not guarded")
	}
	gate.Ready(epoch)
	firstPacket := now.Add(1200 * time.Millisecond)
	for i, expected := range []string{"settling_start", "settling", ""} {
		at := firstPacket.Add(time.Duration(i) * 150 * time.Millisecond)
		if reason := gate.Queue(queue, []byte{3}, at); reason != expected {
			t.Fatalf("i=%d reason=%s", i, reason)
		}
		frame := <-queue
		if frame.Discard != (i < 2) {
			t.Fatalf("discard policy changed after queueing: %+v", frame)
		}
	}
}

func TestAudioInputGateWakeBypassesAndOldFramesDrain(t *testing.T) {
	now := time.Unix(100, 0)
	var gate AudioInputGate
	gate.Configure(true, 300*time.Millisecond)
	queue := make(chan OpusInputFrame, 8)
	old, _ := gate.BeginListen("auto", now, queue)
	gate.Ready(old)
	gate.Queue(queue, []byte{1}, now)
	gate.PlaybackStopped(now, true)
	gate.Wake(now.Add(time.Millisecond))
	current, guarded := gate.BeginListen("auto", now.Add(2*time.Millisecond), queue)
	if guarded || len(queue) != 0 || gate.Allows(old) {
		t.Fatal("wake or old-frame isolation failed")
	}
	gate.Ready(old)
	if gate.Allows(current) {
		t.Fatal("stale Ready activated newer turn")
	}
	gate.Ready(current)
	if gate.Queue(queue, []byte{2}, now.Add(3*time.Millisecond)) != "" {
		t.Fatal("keyword command delayed")
	}
	if (<-queue).Discard {
		t.Fatal("keyword command marked discard")
	}
}

func TestAudioInputGateWaitsForAsrInitialization(t *testing.T) {
	var gate AudioInputGate
	gate.Configure(true, 0)
	queue := make(chan OpusInputFrame, 1)
	epoch, _ := gate.BeginListen("auto", time.Now(), queue)
	if !gate.Starting() {
		t.Fatal("new listen must buffer during initialization")
	}
	gate.Queue(queue, []byte{1}, time.Now())
	frame := <-queue
	done := make(chan bool, 1)
	go func() { done <- gate.WaitFrame(context.Background(), frame.Epoch) }()
	gate.Ready(epoch)
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("first buffered command lost")
		}
	case <-time.After(time.Second):
		t.Fatal("ready did not release frame")
	}
	next, _ := gate.BeginListen("auto", time.Now(), queue)
	cancelled := make(chan bool, 1)
	go func() { cancelled <- gate.WaitFrame(context.Background(), next) }()
	gate.Block()
	select {
	case ok := <-cancelled:
		if ok {
			t.Fatal("blocked frame accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("Block did not release waiter")
	}
}

func TestAudioInputGateManualRealtimeDisabledAndLateSpeech(t *testing.T) {
	now := time.Unix(100, 0)
	for _, mode := range []string{"manual", "realtime"} {
		var gate AudioInputGate
		gate.Configure(true, 300*time.Millisecond)
		queue := make(chan OpusInputFrame, 1)
		gate.PlaybackStopped(now, true)
		epoch, guarded := gate.BeginListen(mode, now, queue)
		gate.Block()
		if guarded || gate.Queue(queue, []byte{1}, now) != "" || !gate.Allows(epoch) {
			t.Fatalf("mode %s behavior changed", mode)
		}
	}
	var gate AudioInputGate
	gate.Configure(true, 300*time.Millisecond)
	queue := make(chan OpusInputFrame, 1)
	gate.PlaybackStopped(now, true)
	epoch, _ := gate.BeginListen("auto", now, queue)
	gate.Ready(epoch)
	gate.Queue(queue, []byte{1}, now.Add(2*time.Second))
	if (<-queue).Discard {
		t.Fatal("late first real word discarded")
	}
	gate.Configure(false, 300*time.Millisecond)
	gate.Block()
	if gate.Queue(queue, []byte{2}, now) != "" {
		t.Fatal("disabled gate blocked")
	}
}

func TestAudioInputGateConcurrentLifecycle(t *testing.T) {
	var gate AudioInputGate
	gate.Configure(true, 300*time.Millisecond)
	queue := make(chan OpusInputFrame, 100)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				epoch, _ := gate.BeginListen("auto", time.Now(), queue)
				gate.Ready(epoch)
				gate.Queue(queue, []byte{1}, time.Now())
				gate.Forward(epoch, func() {})
				gate.Block()
			}
		}()
	}
	wg.Wait()
}

func TestAudioInputGateSessionReconfigureStartsClosed(t *testing.T) {
	var gate AudioInputGate
	gate.Configure(true, 0)
	queue := make(chan OpusInputFrame, 1)
	epoch, _ := gate.BeginListen("auto", time.Now(), queue)
	gate.Ready(epoch)
	gate.Configure(true, 300*time.Millisecond)
	if gate.Allows(epoch) || gate.Queue(queue, []byte{1}, time.Now()) != "busy" {
		t.Fatal("new session accepted old listen state")
	}
}
