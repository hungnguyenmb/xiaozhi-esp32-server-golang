package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	audio "xiaozhi-esp32-server-golang/internal/data/audio"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/audio/capture"
)

type captureTestConn struct {
	sessionCloseTestConn
	sendErr error
}

func (c *captureTestConn) RecvAudio(context.Context, int) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}
func (c *captureTestConn) SendAudio([]byte) error { return c.sendErr }

func TestAudioCaptureBeforeChatGatesAndSuccessfulOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	root := t.TempDir()
	viper.Set("chat.audio_capture.enabled", true)
	viper.Set("chat.audio_capture.directory", root)
	conn := &captureTestConn{}
	state := &ClientState{DeviceID: "probe", SessionID: "test", VoiceStatus: VoiceStatus{VoiceStop: true}}
	transport := NewServerTransport(conn, state)
	format := audio.AudioFormat{Format: "opus", SampleRate: 16000, Channels: 1, FrameDuration: 60}
	transport.audioCapture.SetFormat("mic", format)
	transport.audioCapture.SetFormat("speaker", format)
	if _, err := transport.RecvAudio(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	conn.sendErr = errors.New("send failed")
	if err := transport.SendAudio([]byte{4}); err == nil {
		t.Fatal("send error lost")
	}
	conn.sendErr = nil
	if err := transport.SendAudio([]byte{5}); err != nil {
		t.Fatal(err)
	}
	transport.Close()
	files, _ := filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	f, _ := os.Open(files[0])
	defer f.Close()
	decoder := json.NewDecoder(f)
	mic, speaker, failed := 0, 0, 0
	for {
		var r capture.Record
		err := decoder.Decode(&r)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if r.Kind == "speaker_send_error" {
			failed++
		}
		if r.Kind == "audio" {
			if r.Direction == "mic" {
				mic++
			} else {
				speaker++
				if len(r.Data) != 1 || r.Data[0] != 5 {
					t.Fatal("unsent audio archived as sent")
				}
			}
		}
	}
	if mic != 1 || speaker != 1 || failed != 1 {
		t.Fatalf("mic=%d speaker=%d failure=%d", mic, speaker, failed)
	}
}
