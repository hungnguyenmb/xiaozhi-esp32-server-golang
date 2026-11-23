package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/hraban/opus.v2"
	audio "xiaozhi-esp32-server-golang/internal/data/audio"
	capture "xiaozhi-esp32-server-golang/internal/domain/audio/capture"
)

func TestExportDecodesOpusAndPreservesTimeline(t *testing.T) {
	dir := t.TempDir()
	encoder, err := opus.NewEncoder(16000, 1, opus.AppAudio)
	if err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 1275)
	n, err := encoder.Encode(make([]int16, 960), packet)
	if err != nil {
		t.Fatal(err)
	}
	anchor := time.Now().UTC()
	f, err := os.Create(filepath.Join(dir, "000001.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	jsonWriter := json.NewEncoder(f)
	jsonWriter.Encode(capture.Record{Kind: "header", Timestamp: anchor})
	format := audio.AudioFormat{Format: "opus", SampleRate: 16000, Channels: 1, FrameDuration: 60}
	for i := 0; i < 2; i++ {
		jsonWriter.Encode(capture.Record{Kind: "audio", Timestamp: anchor.Add(time.Duration(i) * time.Second), Direction: "mic", Format: format, Data: packet[:n]})
	}
	jsonWriter.Encode(capture.Record{Kind: "close"})
	f.Close()
	out := filepath.Join(dir, "timeline.wav")
	if err := export(dir, out, "mic", true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if string(data[:4]) != "RIFF" || binary.LittleEndian.Uint32(data[24:]) != 16000 || len(data) != 44+16960*2 {
		t.Fatalf("WAV length/header invalid: %d", len(data))
	}
	if err := export(dir, out, "mic", true); err == nil {
		t.Fatal("must not overwrite WAV")
	}
	compact := filepath.Join(dir, "compact.wav")
	if err := export(dir, compact, "mic", false); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(compact)
	if len(data) != 44+1920*2 {
		t.Fatalf("compact length: %d", len(data))
	}
}

func TestExportRejectsTruncatedArchive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001.jsonl"), []byte(`{"kind":`), 0600)
	if err := export(dir, filepath.Join(dir, "out.wav"), "mic", true); err == nil {
		t.Fatal("corrupted archive accepted")
	}
}
