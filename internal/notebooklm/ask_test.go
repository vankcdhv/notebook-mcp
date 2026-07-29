package notebooklm

import (
	"context"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

// fakeAskRPC records the source IDs each ask variant ends up sending and serves
// a two-source notebook to ListSources.
type fakeAskRPC struct {
	askSourceIDs    []string
	streamSourceIDs []string
}

func (f *fakeAskRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	return []any{
		[]any{nil, []any{
			[]any{[]any{"source-1"}, "First source"},
			[]any{[]any{"source-2"}, "Second source"},
		}},
	}, nil
}

func (f *fakeAskRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	f.askSourceIDs = sourceIDs
	return "answer", "conv-1", nil, nil
}

func (f *fakeAskRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	f.streamSourceIDs = sourceIDs
	return []rpc.ChatChunk{{Seq: 1, Text: "answer", IsFinal: true}}, "conv-1", nil, nil
}

func (f *fakeAskRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestAskVariantsDefaultToAllSources(t *testing.T) {
	fake := &fakeAskRPC{}
	client := New(fake)

	if _, err := client.Ask(context.Background(), "nb", "question", nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AskStream(context.Background(), "nb", "question", nil, ""); err != nil {
		t.Fatal(err)
	}

	want := []string{"source-1", "source-2"}
	assertSourceIDs(t, "Ask", fake.askSourceIDs, want)
	assertSourceIDs(t, "AskStream", fake.streamSourceIDs, want)
}

func TestAskVariantsKeepExplicitScope(t *testing.T) {
	fake := &fakeAskRPC{}
	client := New(fake)
	scoped := []string{"source-2"}

	if _, err := client.Ask(context.Background(), "nb", "question", scoped, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AskStream(context.Background(), "nb", "question", scoped, ""); err != nil {
		t.Fatal(err)
	}

	assertSourceIDs(t, "Ask", fake.askSourceIDs, scoped)
	assertSourceIDs(t, "AskStream", fake.streamSourceIDs, scoped)
}

func assertSourceIDs(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s source IDs = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s source IDs = %v, want %v", name, got, want)
		}
	}
}
