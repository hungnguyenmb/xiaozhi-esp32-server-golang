package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	data_client "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
)

func TestLLMProviderErrorNeverReachesTTSOrHistoryAndNextTurnWorks(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint("stream=", stream), func(t *testing.T) {
			var failing atomic.Bool
			failing.Store(true)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if failing.Load() {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `{"error":{"message":"TECHNICAL_TEST_ERROR missing tool_call_id call_test","type":"invalid_request_error"}}`)
					return
				}
				var req struct {
					Stream bool `json:"stream"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.Stream {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: {\"id\":\"unit\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Xin chào.\"}}]}\n\ndata: {\"id\":\"unit\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
				} else {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"id":"unit","choices":[{"index":0,"message":{"role":"assistant","content":"Xin chào."},"finish_reason":"stop"}]}`)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			fullText := &strings.Builder{}
			ctx = context.WithValue(ctx, fullTextKey, fullText)
			manager := newTestLLMManager(data_client.MemoryModeShort)
			manager.clientState.Ctx = ctx
			manager.clientState.DeviceConfig.Llm.Provider = "openai"
			manager.clientState.DeviceConfig.Llm.Config = map[string]interface{}{"type": "openai", "model_name": "unit-test", "api_key": "unit-test", "base_url": server.URL + "/v1", "streamable": stream}
			manager.lastMessageID = make(map[string]string)
			manager.ttsManager = newTestTTSManager(false)
			user := schema.UserMessage("xin chào")
			responses, err := manager.handleLLMWithContextAndTools(ctx, []*schema.Message{user}, nil)
			if err != nil {
				t.Fatal(err)
			}
			result, err := manager.handleLLMResponse(ctx, user, responses)
			if err != nil || !result.ok {
				t.Fatalf("error turn did not terminate: %v", err)
			}
			if manager.ttsManager.ttsQueueSeq.Load() != 0 {
				t.Fatal("technical error was queued for speech")
			}
			if fullText.Len() != 0 || len(manager.clientState.Dialogue.Messages) != 0 {
				t.Fatal("technical error contaminated assistant history")
			}
			failing.Store(false)
			responses, err = manager.handleLLMWithContextAndTools(ctx, []*schema.Message{user}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var normal strings.Builder
			ended := false
			for response := range responses {
				normal.WriteString(response.Text)
				ended = ended || response.IsEnd
			}
			if !strings.Contains(normal.String(), "Xin chào") || !ended {
				t.Fatalf("next turn broken: %q ended=%t", normal.String(), ended)
			}
		})
	}
}

func TestToolHistoryPersistenceIDsAreUniqueForRapidResults(t *testing.T) {
	manager := newTestLLMManager(data_client.MemoryModeShort)
	ids := map[string]bool{}
	listener := func(e *eventbus.AddMessageEvent) {
		if e.ClientState != manager.clientState {
			return
		}
		if ids[e.MessageID] {
			t.Errorf("tool history ID collision: %s", e.MessageID)
		}
		ids[e.MessageID] = true
	}
	if err := eventbus.Get().Subscribe(eventbus.TopicAddMessage, listener); err != nil {
		t.Fatal(err)
	}
	defer eventbus.Get().Unsubscribe(eventbus.TopicAddMessage, listener)
	for i := 0; i < 20; i++ {
		if err := manager.AddLlmMessage(context.Background(), schema.ToolMessage("result", fmt.Sprint("call_", i))); err != nil {
			t.Fatal(err)
		}
	}
	if len(ids) != 20 {
		t.Fatalf("persisted %d unique IDs for 20 tool results", len(ids))
	}
}

func TestGetMessagesRepairsLegacyMultiToolRoundInBothMemoryModes(t *testing.T) {
	for _, mode := range []string{data_client.MemoryModeShort, data_client.MemoryModeNone} {
		t.Run(mode, func(t *testing.T) {
			manager := newTestLLMManager(mode)
			assistant := schema.AssistantMessage("", []schema.ToolCall{
				{ID: "a", Type: "function", Function: schema.FunctionCall{Name: "first", Arguments: "{}"}},
				{ID: "b", Type: "function", Function: schema.FunctionCall{Name: "second", Arguments: "{}"}},
			})
			history := []*schema.Message{schema.UserMessage("hello"), assistant, assistant,
				schema.ToolMessage("b-result", "b"), schema.ToolMessage("a-result", "a")}
			ctx := context.Background()
			if mode == data_client.MemoryModeNone {
				ctx = appendToolRoundMessagesToContext(ctx, history)
			} else {
				for _, m := range history {
					manager.clientState.AddMessage(m)
				}
			}
			messages := manager.GetMessages(ctx, nil, 20, nil)
			if len(messages) != 5 || messages[2].Role != schema.Assistant || len(messages[2].ToolCalls) != 2 ||
				messages[3].ToolCallID != "a" || messages[4].ToolCallID != "b" {
				t.Fatal("invalid provider request after legacy multi-tool round", messages)
			}
		})
	}
}

func TestLLMProviderStreamErrorPreservesLegitimatePartialText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"unit\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Xin chào.\"}}]}\n\n")
		w.(http.Flusher).Flush()
		// The provider must emit a structured error for a malformed later chunk.
		fmt.Fprint(w, "data: {TECHNICAL_STREAM_FAILURE}\n\n")
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager := newTestLLMManager(data_client.MemoryModeShort)
	manager.clientState.DeviceConfig.Llm.Provider = "openai"
	manager.clientState.DeviceConfig.Llm.Config = map[string]interface{}{"type": "openai", "model_name": "unit-test",
		"api_key": "unit-test", "base_url": server.URL + "/v1", "streamable": true}
	responses, err := manager.handleLLMWithContextAndTools(ctx, []*schema.Message{schema.UserMessage("hello")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	ended := false
	for response := range responses {
		text.WriteString(response.Text)
		ended = ended || response.IsEnd
	}
	if text.String() != "Xin chào." || !ended {
		t.Fatalf("partial text/error boundary: %q ended=%t", text.String(), ended)
	}
}
