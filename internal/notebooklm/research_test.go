package notebooklm

import (
	"context"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

// fakeResearchRPC replays the poll payload NotebookLM returns:
// [[[taskID, [notebookID, [query, sourceType], _, [[[url, title, snippet], ...]], status], ...]]].
type fakeResearchRPC struct{}

func (f *fakeResearchRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	found := []any{
		[]any{"https://example.com/one", "First result", "snippet one", float64(1)},
		[]any{"https://example.com/two", "Second result", "snippet two", float64(1)},
	}
	task := []any{
		"task-1",
		[]any{"nb", []any{"query", float64(1)}, float64(1), []any{found}, float64(2)},
		[]any{float64(1785344748), float64(246650000)},
	}
	return []any{[]any{task}}, nil
}

func (f *fakeResearchRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "", "", nil, nil
}

func (f *fakeResearchRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return nil, "", nil, nil
}

func (f *fakeResearchRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestPollResearchReturnsFoundSources(t *testing.T) {
	results, err := New(&fakeResearchRPC{}).PollResearch(context.Background(), "nb")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	first := results[0]
	if first.TaskID != "task-1" || first.Status != "completed" {
		t.Fatalf("first = %+v", first)
	}
	if first.URL != "https://example.com/one" || first.Title != "First result" {
		t.Fatalf("first = %+v", first)
	}
	if results[1].URL != "https://example.com/two" {
		t.Fatalf("second = %+v", results[1])
	}
}
