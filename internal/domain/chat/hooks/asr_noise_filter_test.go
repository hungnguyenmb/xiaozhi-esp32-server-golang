package hooks

import (
	"context"
	"testing"

	"github.com/spf13/viper"
)

func TestNoiseHookDropsOneResultButKeepsFollowingVietnameseSpeech(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(asrNoiseFilterConfigPath+".enabled", true)
	enabled := true
	hub := NewHub(context.Background())
	t.Cleanup(hub.Close)
	if err := RegisterBuiltinPlugins(hub, map[string]BuiltinPluginConfig{
		"asr_noise_filter": {Enabled: &enabled},
	}); err != nil {
		t.Fatal(err)
	}
	ctx := Context{Ctx: context.Background(), DeviceID: "test-device"}
	if _, stop, err := hub.EmitASROutput(ctx, ASROutputData{Text: "...", VoiceDurationMs: 80}); err != nil || !stop {
		t.Fatalf("punctuation should stop this result: stop=%v err=%v", stop, err)
	}
	for _, text := range []string{"dừng", "Xin chào, bạn nghe rõ không?"} {
		got, stop, err := hub.EmitASROutput(ctx, ASROutputData{Text: text, VoiceDurationMs: 80})
		if err != nil || stop || got.Text != text {
			t.Errorf("next utterance %q: stop=%v payload=%q err=%v", text, stop, got.Text, err)
		}
	}
}

func TestShouldDropASRNoiseTextDropsPunctuationOnly(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = true

	drop, reason := ShouldDropASRNoiseText("。。。？！", 0, cfg)

	if !drop {
		t.Fatalf("ShouldDropASRNoiseText() drop = false, want true")
	}
	if reason != "punctuation_only" {
		t.Fatalf("ShouldDropASRNoiseText() reason = %q, want punctuation_only", reason)
	}
}

func TestShouldDropASRNoiseTextDropsConfiguredPhrasesAfterNormalization(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = true

	cases := []string{
		"嗯。",
		"啊",
		"标识符号",
		"标点符号。",
	}

	for _, text := range cases {
		drop, reason := ShouldDropASRNoiseText(text, 120, cfg)
		if !drop {
			t.Fatalf("ShouldDropASRNoiseText(%q) drop = false, want true", text)
		}
		if reason != "drop_phrase" {
			t.Fatalf("ShouldDropASRNoiseText(%q) reason = %q, want drop_phrase", text, reason)
		}
	}
}

func TestShouldDropASRNoiseTextKeepsConfiguredShortCommands(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = true

	for _, text := range []string{"好。", "停", "开", "关"} {
		drop, reason := ShouldDropASRNoiseText(text, 80, cfg)
		if drop {
			t.Fatalf("ShouldDropASRNoiseText(%q) drop = true reason=%q, want false", text, reason)
		}
	}
}

func TestShouldDropASRNoiseTextKeepsNoiseWhenDisabled(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = false

	drop, reason := ShouldDropASRNoiseText("嗯。", 80, cfg)

	if drop {
		t.Fatalf("ShouldDropASRNoiseText() drop = true reason=%q, want false", reason)
	}
	if reason != "disabled" {
		t.Fatalf("ShouldDropASRNoiseText() reason = %q, want disabled", reason)
	}
}

func TestVietnameseShortCommandsAndAcknowledgementsAreKept(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = true
	for _, text := range []string{"DỪNG!", "tắt", "bật", "vâng", "dạ", "không", "có", "ừ", "ờ", "ơi"} {
		if drop, reason := ShouldDropASRNoiseText(text, 80, cfg); drop || reason != "keep_phrase" {
			t.Errorf("text=%q drop=%v reason=%s", text, drop, reason)
		}
	}
}

func TestShortFillerRequiresKnownBriefVoiceDuration(t *testing.T) {
	cfg := DefaultASRNoiseFilterConfig()
	cfg.Enabled = true
	for _, ms := range []int64{0, 300, 500} {
		if drop, _ := ShouldDropASRNoiseText("hmm", ms, cfg); drop {
			t.Errorf("should keep filler when duration is unknown or not brief: %d", ms)
		}
	}
	if drop, reason := ShouldDropASRNoiseText("hmm", 120, cfg); !drop || reason != "short_filler" {
		t.Fatalf("brief filler: drop=%v reason=%s", drop, reason)
	}
	for _, text := range []string{"ăn", "ba", "gọi mẹ", "tôi đang nói chuyện", "hmm tôi muốn hỏi"} {
		if drop, reason := ShouldDropASRNoiseText(text, 80, cfg); drop {
			t.Errorf("valid speech %q dropped: %s", text, reason)
		}
	}
}
