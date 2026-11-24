// Replay private WAVs or explicit capture listen windows through the same
// frontend and turn assembler as live input. Never calls LLM/TTS or writes DB.
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-audio/wav"
	"github.com/gorilla/websocket"
	"gopkg.in/hraban/opus.v2"
	"xiaozhi-esp32-server-golang/internal/domain/audio/capture"
	"xiaozhi-esp32-server-golang/internal/domain/audio/frontend"
)

type options struct {
	input, windows, wav, suite, asr string
	denoise, text                   bool
	start, low, strong              float64
	silence                         int
	tail                            int
}
type window struct {
	Start, End time.Time
	GuardMs    int `json:"guard_ms"`
}
type segment struct {
	StartMs, EndMs int64
	Samples        int
	Open           bool
	ASRMs          float64 `json:",omitempty"`
	Text           *string `json:",omitempty"`
}
type report struct {
	ID                                                           string
	Denoise                                                      bool
	InputSamples, OutputSamples, VoiceHops, Candidates, Accepted int
	PaddingSamples                                               int
	NativeDelayMs, ProcessingMs, AudioMs                         float64
	Segments                                                     []segment
}

func main() {
	var o options
	flag.StringVar(&o.input, "input", "", "capture directory, requires -windows")
	flag.StringVar(&o.windows, "windows", "", "private JSON [{Start,End,guard_ms}], timestamps RFC3339")
	flag.StringVar(&o.wav, "wav", "", "one PCM16/16k/mono WAV")
	flag.StringVar(&o.suite, "suite", "", "private JSON [{id,wav}], paths relative to suite")
	flag.StringVar(&o.asr, "asr-url", "", "optional FunASR-compatible WebSocket, never LLM/TTS")
	flag.BoolVar(&o.denoise, "denoise", true, "RNNoise, false isolates segmentation without denoise")
	flag.BoolVar(&o.text, "include-text", false, "include private ASR text in JSON output")
	flag.Float64Var(&o.start, "start", .55, "onset probability")
	flag.Float64Var(&o.low, "continue", .35, "speech maintenance probability")
	flag.Float64Var(&o.strong, "strong", .85, "single-hop strong onset")
	flag.IntVar(&o.tail, "tail-ms", 0, "WAV-only synthetic silence to drain DSP; reported separately, not endpoint latency")
	flag.IntVar(&o.silence, "silence-ms", 500, "strict endpoint silence threshold; use effective live value")
	flag.Parse()
	reports, err := run(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reports); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type replay struct {
	p        *frontend.Processor
	turn     *frontend.Turn
	r        report
	o        options
	position int64
	pcm      []float32
	begin    int64
}

func newReplay(id string, o options) (*replay, error) {
	p, err := frontend.New(frontend.Config{Denoise: o.denoise, Start: float32(o.start), Continue: float32(o.low), Strong: float32(o.strong)})
	if err != nil {
		return nil, err
	}
	return &replay{p: p, turn: frontend.NewTurn(o.silence), r: report{ID: id, Denoise: o.denoise, NativeDelayMs: p.NativeDelayMs()}, o: o}, nil
}
func (r *replay) push(pcm []float32) error {
	start := time.Now()
	frames, err := r.p.Process(pcm)
	r.r.ProcessingMs += float64(time.Since(start)) / float64(time.Millisecond)
	if err != nil {
		return err
	}
	r.r.InputSamples += len(pcm)
	for _, f := range frames {
		f = r.p.Classify(f)
		r.r.OutputSamples += len(f.PCM)
		r.position += int64(len(f.PCM))
		if f.Voice {
			r.r.VoiceHops++
		}
		if f.Event == "candidate" {
			r.r.Candidates++
		}
		if f.Event == "accepted" {
			r.r.Accepted++
		}
		out := r.turn.Push(f)
		if out.Started {
			r.begin = r.position - int64(len(out.PCM))
		}
		r.pcm = append(r.pcm, out.PCM...)
		if len(r.pcm) > frontend.SampleRate*60 {
			return errors.New("segment exceeds 60 seconds; check endpoint")
		}
		if out.Ended {
			if err := r.finish(false); err != nil {
				return err
			}
			r.turn.End()
			r.p.EndTurn()
		}
	}
	return nil
}
func (r *replay) finish(open bool) error {
	if len(r.pcm) == 0 {
		return nil
	}
	s := segment{StartMs: r.begin * 1000 / frontend.SampleRate, EndMs: r.position * 1000 / frontend.SampleRate, Samples: len(r.pcm), Open: open}
	if r.o.asr != "" {
		start := time.Now()
		text, err := recognize(r.o.asr, r.pcm)
		if err != nil {
			return err
		}
		s.ASRMs = float64(time.Since(start)) / float64(time.Millisecond)
		if r.o.text {
			s.Text = &text
		}
	}
	r.r.Segments = append(r.r.Segments, s)
	r.pcm = nil
	return nil
}
func (r *replay) close() (report, error) {
	defer r.p.Close()
	err := r.finish(true)
	r.r.AudioMs = float64(r.r.InputSamples-r.r.PaddingSamples) * 1000 / frontend.SampleRate
	return r.r, err
}
func run(o options) ([]report, error) {
	selected := 0
	for _, s := range []string{o.input, o.wav, o.suite} {
		if s != "" {
			selected++
		}
	}
	if selected != 1 {
		return nil, errors.New("choose exactly one of -input, -wav, -suite")
	}
	if o.tail < 0 || o.tail > 5000 {
		return nil, errors.New("tail-ms must be 0..5000")
	}
	if o.input != "" && o.tail != 0 {
		return nil, errors.New("do not add artificial tail to captured listen windows")
	}
	if o.silence < 1 {
		return nil, errors.New("silence-ms must be positive")
	}
	if o.input != "" {
		return replayCapture(o)
	}
	items := []struct{ ID, Wav string }{{"wav", o.wav}}
	if o.suite != "" {
		b, err := os.ReadFile(o.suite)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &items); err != nil {
			return nil, err
		}
		for i := range items {
			if !filepath.IsAbs(items[i].Wav) {
				items[i].Wav = filepath.Join(filepath.Dir(o.suite), items[i].Wav)
			}
		}
	}
	var reports []report
	for _, item := range items {
		pcm, err := readWav(item.Wav)
		if err != nil {
			return nil, err
		}
		r, err := newReplay(item.ID, o)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(pcm); i += 960 {
			end := i + 960
			if end > len(pcm) {
				end = len(pcm)
			}
			if err = r.push(pcm[i:end]); err != nil {
				r.p.Close()
				return nil, err
			}
		}
		if o.tail > 0 {
			padding := o.tail * frontend.SampleRate / 1000
			if err = r.push(make([]float32, padding)); err != nil {
				r.p.Close()
				return nil, err
			}
			r.r.PaddingSamples = padding
		}
		result, err := r.close()
		if err != nil {
			return nil, err
		}
		reports = append(reports, result)
	}
	return reports, nil
}
func readWav(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > 32*1024*1024 {
		return nil, errors.New("WAV too large (32 MiB maximum)")
	}
	d := wav.NewDecoder(f)
	d.ReadInfo()
	if d.SampleRate != 16000 || d.NumChans != 1 || d.BitDepth != 16 {
		return nil, fmt.Errorf("need PCM16/16k/mono WAV: %s", path)
	}
	b, err := d.FullPCMBuffer()
	if err != nil {
		return nil, err
	}
	out := make([]float32, len(b.Data))
	for i, x := range b.Data {
		out[i] = float32(x) / 32768
	}
	return out, nil
}
func replayCapture(o options) ([]report, error) {
	if o.windows == "" {
		return nil, errors.New("capture requires explicit listen windows; replaying busy audio would be misleading")
	}
	b, err := os.ReadFile(o.windows)
	if err != nil {
		return nil, err
	}
	var windows []window
	if err = json.Unmarshal(b, &windows); err != nil {
		return nil, err
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].Start.Before(windows[j].Start) })
	for i, w := range windows {
		if !w.End.After(w.Start) || w.GuardMs < 0 || w.GuardMs > 1000 || (i > 0 && w.Start.Before(windows[i-1].End)) {
			return nil, errors.New("invalid/overlapping listen window")
		}
	}
	// Bounded metadata scan. Read each segment again per window to avoid loading
	// full private recordings into memory; files are never modified.
	files, err := filepath.Glob(filepath.Join(o.input, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no capture segments")
	}
	var reports []report
	for index, w := range windows {
		r, err := newReplay(fmt.Sprintf("window-%03d", index+1), o)
		if err != nil {
			return nil, err
		}
		err = func() error {
			defer r.p.Close()
			decoder, err := opus.NewDecoder(16000, 1)
			if err != nil {
				return err
			}
			var first time.Time
			var sequence uint64
			for _, name := range files {
				f, err := os.Open(name)
				if err != nil {
					return err
				}
				err = func() error {
					defer f.Close()
					d := json.NewDecoder(f)
					for {
						var record capture.Record
						if err := d.Decode(&record); err != nil {
							if err == io.EOF {
								return nil
							}
							return err
						}
						if record.Sequence > 0 {
							if sequence > 0 && record.Sequence != sequence+1 {
								return errors.New("capture sequence gap")
							}
							sequence = record.Sequence
						}
						if record.Dropped > 0 {
							return errors.New("capture reports dropped records")
						}
						if record.Timestamp.Before(w.Start) || !record.Timestamp.Before(w.End) || record.Kind != "audio" || record.Direction != "mic" {
							continue
						}
						if record.Format.Format != "opus" || record.Format.SampleRate != 16000 || record.Format.Channels != 1 {
							return errors.New("capture format not Opus16k/mono")
						}
						pcm := make([]float32, 1920)
						n, err := decoder.DecodeFloat32(record.Data, pcm)
						if err != nil {
							return err
						}
						if first.IsZero() {
							first = record.Timestamp
						}
						// Match gate: guard starts with the first packet only when it arrives
						// within one second of listen-start. Decoder is primed during guard.
						guarded := w.GuardMs > 0 && first.Sub(w.Start) <= time.Second && record.Timestamp.Sub(first) < time.Duration(w.GuardMs)*time.Millisecond
						if !guarded {
							if err := r.push(pcm[:n]); err != nil {
								return err
							}
						}
					}
				}()
				if err != nil {
					return err
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		result, err := r.close()
		if err != nil {
			return nil, err
		}
		reports = append(reports, result)
	}
	return reports, nil
}
func recognize(url string, pcm []float32) (string, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	_ = conn.SetWriteDeadline(deadline)
	_ = conn.SetReadDeadline(deadline)
	if err = conn.WriteJSON(map[string]any{"mode": "offline", "audio_fs": 16000, "wav_format": "pcm", "wav_name": "frontend-replay", "is_speaking": true, "chunk_size": []int{5, 10, 5}, "chunk_interval": 10}); err != nil {
		return "", err
	}
	for i := 0; i < len(pcm); i += 960 {
		end := i + 960
		if end > len(pcm) {
			end = len(pcm)
		}
		data := make([]byte, (end-i)*2)
		for j, x := range pcm[i:end] {
			if x > 32767.0/32768 {
				x = 32767.0 / 32768
			}
			if x < -1 {
				x = -1
			}
			binary.LittleEndian.PutUint16(data[2*j:], uint16(int16(x*32768)))
		}
		if err = conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			return "", err
		}
	}
	if err = conn.WriteJSON(map[string]any{"is_speaking": false}); err != nil {
		return "", err
	}
	for {
		var response struct {
			Text  string `json:"text"`
			Final bool   `json:"is_final"`
			Mode  string `json:"mode"`
		}
		if err = conn.ReadJSON(&response); err != nil {
			return "", err
		}
		if response.Final || response.Mode == "offline" {
			return response.Text, nil
		}
	}
}
