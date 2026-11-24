package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestReplayRequiresExplicitInputAndListenWindows(t *testing.T) {
	if _, err := run(options{}); err == nil {
		t.Fatal("missing source accepted")
	}
	if _, err := run(options{input: "private", silence: 400}); err == nil {
		t.Fatal("capture without windows accepted")
	}
	if _, err := run(options{wav: "a", suite: "b", silence: 400}); err == nil {
		t.Fatal("ambiguous sources accepted")
	}
}
func TestASRReplayWireAndSaturation(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		var start map[string]any
		if err = c.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		if start["is_speaking"] != true || start["audio_fs"] != float64(16000) {
			t.Error("incorrect ASR start")
		}
		kind, data, err := c.ReadMessage()
		if err != nil {
			t.Error(err)
			return
		}
		if kind != websocket.BinaryMessage || len(data) != 4 || data[0] != 255 || data[1] != 127 || data[2] != 0 || data[3] != 128 {
			t.Error("PCM saturation failed")
		}
		_, data, err = c.ReadMessage()
		if err != nil {
			t.Error(err)
			return
		}
		var stop map[string]any
		_ = json.Unmarshal(data, &stop)
		if stop["is_speaking"] != false {
			t.Error("missing stop")
		}
		_ = c.WriteJSON(map[string]any{"text": "test", "is_final": true})
	}))
	defer server.Close()
	text, err := recognize("ws"+strings.TrimPrefix(server.URL, "http"), []float32{1, -1})
	if err != nil || text != "test" {
		t.Fatalf("%q %v", text, err)
	}
}
