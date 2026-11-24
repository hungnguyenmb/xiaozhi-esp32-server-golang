package client

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func historyCall(id string) schema.ToolCall {
	return schema.ToolCall{ID: id, Type: "function", Function: schema.FunctionCall{Name: "test_" + id, Arguments: "{}"}}
}
func historyAssistant(ids ...string) *schema.Message {
	var calls []schema.ToolCall
	for _, id := range ids {
		calls = append(calls, historyCall(id))
	}
	return schema.AssistantMessage("", calls)
}
func historyShape(messages []*schema.Message) []string {
	var out []string
	for _, m := range messages {
		s := string(m.Role)
		if m.Role == schema.Assistant && len(m.ToolCalls) > 0 {
			var ids []string
			for _, c := range m.ToolCalls {
				ids = append(ids, c.ID)
			}
			s += ":" + strings.Join(ids, ",")
		}
		if m.Role == schema.Tool {
			s += ":" + m.ToolCallID
		}
		out = append(out, s)
	}
	return out
}
func TestAlignToolMessagesContiguousRounds(t *testing.T) {
	a, b := schema.ToolMessage("result-a", "a"), schema.ToolMessage("result-b", "b")
	text := schema.AssistantMessage("Tôi sẽ kiểm tra.", []schema.ToolCall{historyCall("a")})
	cases := []struct {
		name string
		in   []*schema.Message
		want []string
	}{
		{"two calls exactly once", []*schema.Message{historyAssistant("a", "b"), a, b}, []string{"assistant:a,b", "tool:a", "tool:b"}},
		{"repair adjacent legacy duplicate", []*schema.Message{historyAssistant("a", "b"), historyAssistant("a", "b"), a, b}, []string{"assistant:a,b", "tool:a", "tool:b"}},
		{"partial drops only unanswered call", []*schema.Message{historyAssistant("a", "b"), a}, []string{"assistant:a", "tool:a"}},
		{"orphan result at truncated start", []*schema.Message{a, b, schema.UserMessage("next")}, []string{"user"}},
		{"cannot borrow across user", []*schema.Message{historyAssistant("a"), schema.UserMessage("next"), a}, []string{"user"}},
		{"no response keeps ordinary assistant text", []*schema.Message{text, schema.UserMessage("next")}, []string{"assistant", "user"}},
		{"two rounds never merge", []*schema.Message{historyAssistant("a"), a, historyAssistant("b"), b}, []string{"assistant:a", "tool:a", "assistant:b", "tool:b"}},
		{"out of order tool results", []*schema.Message{historyAssistant("a", "b"), b, a}, []string{"assistant:a,b", "tool:a", "tool:b"}},
		{"duplicate IDs and results", []*schema.Message{historyAssistant("a", "a", ""), a, a}, []string{"assistant:a", "tool:a"}},
		{"nil within round", []*schema.Message{nil, historyAssistant("a"), nil, a, nil}, []string{"assistant:a", "tool:a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, _ := json.Marshal(tc.in)
			got := AlignToolMessages(tc.in)
			if shape := historyShape(got); !reflect.DeepEqual(shape, tc.want) {
				t.Fatalf("shape=%v want=%v", shape, tc.want)
			}
			after, _ := json.Marshal(tc.in)
			if string(before) != string(after) {
				t.Fatal("history source mutated")
			}
			if twice := AlignToolMessages(got); !reflect.DeepEqual(got, twice) {
				t.Fatal("alignment not idempotent")
			}
		})
	}
}
func TestAlignToolMessagesAfterInitAndHistoryTruncation(t *testing.T) {
	state := &ClientState{Dialogue: &Dialogue{}}
	input := []*schema.Message{historyAssistant("a", "b"), historyAssistant("a", "b"), schema.ToolMessage("a", "a"), schema.ToolMessage("b", "b"), schema.UserMessage("next")}
	if err := state.InitMessages(input); err != nil {
		t.Fatal(err)
	}
	if got := historyShape(state.GetMessages(20)); !reflect.DeepEqual(got, []string{"assistant:a,b", "tool:a", "tool:b", "user"}) {
		t.Fatal(got)
	}
	if got := historyShape(state.GetMessages(3)); !reflect.DeepEqual(got, []string{"user"}) {
		t.Fatal("truncated orphan tools survived", got)
	}
}
