package chat

import (
	"time"
	client "xiaozhi-esp32-server-golang/internal/data/client"
	log "xiaozhi-esp32-server-golang/logger"
)

// An empty/transient result has no assistant playback or client listen/start.
// Reopen ASR admission ourselves rather than waiting forever for a TTS stop.
func restartASRInputForRetry(state *client.ClientState, restart func() error) error {
	auto := state.ListenMode == "auto"
	var epoch uint64
	if auto {
		epoch, _ = state.AudioInput.BeginListen("auto", time.Now(), state.OpusAudioBuffer)
	}
	if err := restart(); err != nil {
		if auto {
			state.AudioInput.Block()
		}
		return err
	}
	if auto {
		state.AudioInput.Ready(epoch)
		log.Infof("ASR_INPUT_GATE retry_opened: device=%s epoch=%d", state.DeviceID, epoch)
	}
	return nil
}
