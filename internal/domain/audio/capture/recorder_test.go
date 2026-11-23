package capture

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	audio "xiaozhi-esp32-server-golang/internal/data/audio"
)

func TestArchiveKeepsOriginalPacketsAndDirections(t *testing.T) {
	root := t.TempDir()
	r, err := New(root, "robot/test")
	if err != nil {
		t.Fatal(err)
	}
	format := audio.AudioFormat{Format: "opus", SampleRate: 16000, Channels: 1, FrameDuration: 60}
	r.SetFormat("mic", format)
	r.SetFormat("speaker", format)
	packet := []byte{1, 2, 3}
	r.Audio("mic", packet)
	packet[0] = 9
	r.Event("tts_start")
	r.Audio("speaker", []byte{4, 5})
	r.Event("tts_stop")
	r.Close()
	r.Close()
	r.Audio("mic", []byte{9})
	files, _ := filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("segments: %v", files)
	}
	st, _ := os.Stat(files[0])
	if st.Mode().Perm() != 0600 {
		t.Fatal("private file permissions")
	}
	f, _ := os.Open(files[0])
	defer f.Close()
	decoder := json.NewDecoder(f)
	var records []Record
	for {
		var record Record
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if len(records) != 6 || records[0].Kind != "header" || records[5].Kind != "close" {
		t.Fatalf("records: %+v", records)
	}
	if records[1].Direction != "mic" || records[1].Data[0] != 1 || records[1].Format != format || records[3].Direction != "speaker" || records[3].Sequence != 3 {
		t.Fatalf("packet metadata: %+v", records)
	}
}

func TestQueueOverflowIsReportedWithoutBlocking(t *testing.T) {
	r := &Recorder{queue: make(chan Record, 1)}
	r.Audio("mic", []byte{1})
	r.Audio("mic", []byte{2})
	if r.dropped.Load() != 1 || len(r.queue) != 1 {
		t.Fatal("overflow not counted")
	}
}

func TestConcurrentAppendAndClose(t *testing.T) {
	r, err := New(t.TempDir(), "robot")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				r.Audio("mic", []byte{1})
			}
			r.Close()
		}()
	}
	wg.Wait()
}

func TestWriterFailureDoesNotBlockConversation(t *testing.T) {
	r := &Recorder{dir: filepath.Join(t.TempDir(), "missing"), queue: make(chan Record, 1), done: make(chan struct{})}
	go r.run()
	r.Event("probe")
	<-r.done
	if !r.failed.Load() {
		t.Fatal("writer error not exposed")
	}
	r.Audio("mic", []byte{1})
	r.Close()
	if r.dropped.Load() < 2 {
		t.Fatal("failed and rejected packets must be counted")
	}
}

func TestSegmentRotationDoesNotDeleteOrLosePackets(t *testing.T) {
	root := t.TempDir()
	r, err := New(root, "rotation")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		r.Audio("mic", make([]byte, 256*1024))
	}
	r.Close()
	files, _ := filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
	if len(files) < 2 {
		t.Fatalf("expected rotation: %v", files)
	}
	count := 0
	for _, name := range files {
		st, _ := os.Stat(name)
		if st.Size() > segmentBytes {
			t.Fatalf("segment too large: %d", st.Size())
		}
		f, _ := os.Open(name)
		decoder := json.NewDecoder(f)
		for {
			var record Record
			err := decoder.Decode(&record)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if record.Kind == "audio" {
				count++
			}
		}
		f.Close()
	}
	if count != 32 || r.dropped.Load() != 0 {
		t.Fatalf("lost packets count=%d dropped=%d", count, r.dropped.Load())
	}
}
