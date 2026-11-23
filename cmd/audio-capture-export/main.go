// Export one connection's captured Opus packets into a playable PCM16 WAV.
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
	"time"

	"gopkg.in/hraban/opus.v2"
	capture "xiaozhi-esp32-server-golang/internal/domain/audio/capture"
)

func main() {
	input := flag.String("input", "", "one connection capture directory")
	output := flag.String("output", "", "new WAV file (never overwritten)")
	direction := flag.String("direction", "mic", "mic or speaker")
	timeline := flag.Bool("timeline", true, "approximate server timeline, filling gaps over 100ms")
	flag.Parse()
	if err := export(*input, *output, *direction, *timeline); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(input, output, direction string, timeline bool) (result error) {
	if input == "" || output == "" || (direction != "mic" && direction != "speaker") {
		return errors.New("input/output required; direction must be mic or speaker")
	}
	files, err := filepath.Glob(filepath.Join(input, "*.jsonl"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no capture segments")
	}
	out, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	var sampleRate, channels int
	var samples int64
	defer func() {
		if sampleRate > 0 {
			if _, err := out.Seek(0, 0); err == nil {
				if err = writeWAVHeader(out, sampleRate, channels, uint32(samples*2)); result == nil {
					result = err
				}
			} else if result == nil {
				result = err
			}
		}
		if err := out.Close(); result == nil {
			result = err
		}
	}()
	if _, err = out.Write(make([]byte, 44)); err != nil {
		return err
	}
	var decoder *opus.Decoder
	var anchor time.Time
	writeSamples := func(pcm []int16) error {
		if samples+int64(len(pcm)) > (int64(^uint32(0))-36)/2 {
			return errors.New("WAV exceeds 4 GiB; export smaller recording")
		}
		if err := binary.Write(out, binary.LittleEndian, pcm); err != nil {
			return err
		}
		samples += int64(len(pcm))
		return nil
	}
	packets := 0
	closed := false
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		err = func() error {
			defer f.Close()
			records := json.NewDecoder(f)
			for {
				var record capture.Record
				if err := records.Decode(&record); err != nil {
					if err == io.EOF {
						return nil
					}
					return fmt.Errorf("incomplete/invalid segment %s: %w", filepath.Base(name), err)
				}
				if anchor.IsZero() && record.Kind == "header" {
					anchor = record.Timestamp
				}
				if record.Kind == "close" {
					closed = true
				}
				if record.Kind == "close" && record.Dropped > 0 {
					return fmt.Errorf("capture incomplete: %d records dropped", record.Dropped)
				}
				if record.Kind != "audio" || record.Direction != direction {
					continue
				}
				if record.Format.Format != "opus" {
					return fmt.Errorf("unsupported recorded codec %q", record.Format.Format)
				}
				if decoder == nil {
					sampleRate = record.Format.SampleRate
					channels = record.Format.Channels
					if channels < 1 || channels > 2 {
						return errors.New("unsupported channels")
					}
					decoder, err = opus.NewDecoder(sampleRate, channels)
					if err != nil {
						return err
					}
				} else if record.Format.SampleRate != sampleRate || record.Format.Channels != channels {
					return errors.New("audio format changed; split export at format boundary")
				}
				if anchor.IsZero() {
					anchor = record.Timestamp
				}
				pcm := make([]int16, sampleRate*120/1000*channels)
				n, err := decoder.Decode(record.Data, pcm)
				if err != nil {
					return fmt.Errorf("decode sequence %d: %w", record.Sequence, err)
				}
				if timeline {
					target := int64(record.Timestamp.Sub(anchor).Seconds()*float64(sampleRate)) * int64(channels)
					gap := target - samples
					if gap > int64(sampleRate*channels/10) {
						// Write large gaps in bounded chunks, never allocate based on timestamp.
						silence := make([]int16, sampleRate*channels)
						for gap > 0 {
							n := int64(len(silence))
							if n > gap {
								n = gap
							}
							if err := writeSamples(silence[:n]); err != nil {
								return err
							}
							gap -= n
						}
					}
				}
				if err := writeSamples(pcm[:n*channels]); err != nil {
					return err
				}
				packets++
			}
		}()
		if err != nil {
			return err
		}
	}
	if packets == 0 {
		return errors.New("no audio packets for selected direction")
	}
	if !closed {
		fmt.Fprintln(os.Stderr, "Warning: no close marker; recording may be active or interrupted")
	}
	fmt.Printf("Exported %d packets at %d Hz/%d channels; server timeline=%t\n", packets, sampleRate, channels, timeline)
	return nil
}

func writeWAVHeader(w io.Writer, rate, channels int, size uint32) error {
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], size+36)
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(header[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], size)
	_, err := w.Write(header)
	return err
}
