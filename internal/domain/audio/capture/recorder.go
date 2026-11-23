// Package capture archives transport audio without blocking inference on disk I/O.
package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	audio "xiaozhi-esp32-server-golang/internal/data/audio"
	log "xiaozhi-esp32-server-golang/logger"
)

const segmentBytes = 8 * 1024 * 1024

// Record is one JSONL record. Data is base64-encoded by encoding/json.
// Timestamp is server receive/send time, not the physical ADC/DAC time.
type Record struct {
	Kind      string            `json:"kind"`
	Timestamp time.Time         `json:"timestamp"`
	Sequence  uint64            `json:"sequence,omitempty"`
	Device    string            `json:"device,omitempty"`
	Direction string            `json:"direction,omitempty"`
	Format    audio.AudioFormat `json:"format,omitempty"`
	Data      []byte            `json:"data,omitempty"`
	Dropped   uint64            `json:"dropped,omitempty"`
}

type Recorder struct {
	mu            sync.Mutex
	closed        bool
	queue         chan Record
	done          chan struct{}
	failed        atomic.Bool
	dropped       atomic.Uint64
	sequence      uint64
	input, output audio.AudioFormat
	dir, device   string
}

func New(root, device string) (*Recorder, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(root, time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return nil, err
	}
	r := &Recorder{queue: make(chan Record, 1024), done: make(chan struct{}), dir: dir, device: device}
	go r.run()
	return r, nil
}

func (r *Recorder) SetFormat(direction string, format audio.AudioFormat) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if direction == "mic" {
		r.input = format
	} else {
		r.output = format
	}
}

func (r *Recorder) Audio(direction string, data []byte) {
	if r == nil || len(data) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	format := r.input
	if direction == "speaker" {
		format = r.output
	}
	r.enqueue(Record{Kind: "audio", Timestamp: time.Now().UTC(), Direction: direction, Format: format, Data: append([]byte(nil), data...)})
}

func (r *Recorder) Event(kind string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueue(Record{Kind: kind, Timestamp: time.Now().UTC()})
}

// Caller owns mu. Never wait for the disk writer on the audio path.
func (r *Recorder) enqueue(record Record) {
	if r.closed {
		return
	}
	r.sequence++
	record.Sequence = r.sequence
	if !r.failed.Load() {
		select {
		case r.queue <- record:
			return
		default:
		}
	}
	if n := r.dropped.Add(1); n == 1 || n%1000 == 0 {
		log.Warnf("AUDIO_CAPTURE missing records: device=%s dropped=%d writer_failed=%t", r.device, n, r.failed.Load())
	}
}

func (r *Recorder) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.queue)
	}
	r.mu.Unlock()
	<-r.done
}

func (r *Recorder) run() {
	defer close(r.done)
	var file *os.File
	var writer *bufio.Writer
	size, index := 0, 0
	closeFile := func() error {
		if file == nil {
			return nil
		}
		flushErr := writer.Flush()
		closeErr := file.Close()
		file = nil
		if flushErr != nil {
			return flushErr
		}
		return closeErr
	}
	defer func() {
		if err := closeFile(); err != nil {
			r.failed.Store(true)
			log.Errorf("AUDIO_CAPTURE close failed: device=%s err=%v", r.device, err)
		}
	}()
	write := func(record Record) error {
		line, err := json.Marshal(record)
		if err != nil {
			return err
		}
		line = append(line, '\n')
		if file == nil || size+len(line) > segmentBytes {
			free, err := availableBytes(r.dir)
			if err != nil {
				return err
			}
			if free < 2*1024*1024*1024 {
				return fmt.Errorf("less than 2 GiB free; recording stopped without deleting files")
			}
			if err := closeFile(); err != nil {
				return err
			}
			index++
			file, err = os.OpenFile(filepath.Join(r.dir, fmt.Sprintf("%06d.jsonl", index)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			writer = bufio.NewWriterSize(file, 64*1024)
			header, _ := json.Marshal(Record{Kind: "header", Timestamp: time.Now().UTC(), Device: r.device})
			if _, err = writer.Write(append(header, '\n')); err != nil {
				return err
			}
			size = len(header) + 1
		}
		n, err := writer.Write(line)
		size += n
		return err
	}
	fail := func(err error) {
		r.mu.Lock()
		r.failed.Store(true)
		r.dropped.Add(uint64(len(r.queue)))
		r.mu.Unlock()
		log.Errorf("AUDIO_CAPTURE writer failed: device=%s dir=%s err=%v", r.device, r.dir, err)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	log.Infof("AUDIO_CAPTURE opened: device=%s dir=%s", r.device, r.dir)
	for {
		select {
		case record, ok := <-r.queue:
			if !ok {
				if err := write(Record{Kind: "close", Timestamp: time.Now().UTC(), Dropped: r.dropped.Load()}); err != nil {
					fail(err)
				}
				log.Infof("AUDIO_CAPTURE closed: device=%s dir=%s dropped=%d", r.device, r.dir, r.dropped.Load())
				return
			}
			if err := write(record); err != nil {
				r.dropped.Add(1)
				fail(err)
				return
			}
		case <-ticker.C:
			if writer != nil {
				if err := writer.Flush(); err != nil {
					fail(err)
					return
				}
			}
		}
	}
}
