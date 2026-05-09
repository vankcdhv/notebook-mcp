package rpc

import (
	"encoding/json"
	"testing"
)

func TestDecodeChatResponseReturnsReferences(t *testing.T) {
	inner := []any{[]any{
		"answer text",
		nil,
		[]any{"conv-id"},
		nil,
		[]any{nil, nil, nil, []any{
			[]any{[]any{"chunk"}, []any{nil, nil, nil, nil, []any{[]any{[]any{7, 24, []any{[]any{[]any{0, 0, "exact quoted text"}}}}}}, []any{[]any{[]any{"source-1"}}}}},
		}, 1},
	}}
	innerJSON, _ := json.Marshal(inner)
	outerJSON, _ := json.Marshal([]any{[]any{"wrb.fr", nil, string(innerJSON), nil, nil, nil, "generic"}})
	response := ")]}'\n123\n" + string(outerJSON)
	answer, convID, refs, err := DecodeChatResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "answer text" || convID != "conv-id" {
		t.Fatalf("answer=%q convID=%q", answer, convID)
	}
	if len(refs) != 1 {
		t.Fatalf("refs = %d", len(refs))
	}
	if refs[0].SourceID != "source-1" || refs[0].CitedText != "exact quoted text" {
		t.Fatalf("ref = %+v", refs[0])
	}
	if refs[0].StartChar != 7 || refs[0].EndChar != 24 {
		t.Fatalf("positions = %d..%d", refs[0].StartChar, refs[0].EndChar)
	}
}
