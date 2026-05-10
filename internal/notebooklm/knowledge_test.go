package notebooklm

import (
	"context"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

type fakeKnowledgeRPC struct {
	askAnswer string
	convID    string
	calls     []string
}

func (f *fakeKnowledgeRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	f.calls = append(f.calls, method)
	return []any{
		[]any{
			[]any{"source-1"},
			"Source Title",
			[]any{nil, nil, nil, nil, float64(4), nil, nil, []any{"https://example.com"}},
		},
		nil,
		nil,
		[]any{[]any{[]any{"before exact quoted text after"}}},
	}, nil
}

func (f *fakeKnowledgeRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "answer", "conv-1", []rpc.ChatReference{{SourceID: "source-1", CitedText: "exact quoted text"}}, nil
}

func (f *fakeKnowledgeRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return []rpc.ChatChunk{{Seq: 1, Text: "answer", IsFinal: true}}, "conv-1", nil, nil
}

func (f *fakeKnowledgeRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestGetSourceFulltextExtractsContent(t *testing.T) {
	client := New(&fakeKnowledgeRPC{})
	fulltext, err := client.GetSourceFulltext(context.Background(), "nb", "source-1")
	if err != nil {
		t.Fatal(err)
	}
	if fulltext.SourceID != "source-1" {
		t.Fatalf("SourceID = %q", fulltext.SourceID)
	}
	if fulltext.Title != "Source Title" {
		t.Fatalf("Title = %q", fulltext.Title)
	}
	if fulltext.Content != "before exact quoted text after" {
		t.Fatalf("Content = %q", fulltext.Content)
	}
}

func TestKnowledgeSearchReturnsSnippets(t *testing.T) {
	client := New(&fakeKnowledgeRPC{})
	result, err := client.KnowledgeSearch(context.Background(), "nb", "query", nil, 7, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "answer" {
		t.Fatalf("Answer = %q", result.Answer)
	}
	if result.Count != 1 {
		t.Fatalf("Count = %d", result.Count)
	}
	snippet := result.Snippets[0]
	if snippet.SourceID != "source-1" || snippet.SourceTitle != "Source Title" {
		t.Fatalf("snippet source = %+v", snippet)
	}
	if snippet.ExactQuote != "exact quoted text" {
		t.Fatalf("ExactQuote = %q", snippet.ExactQuote)
	}
	if snippet.Snippet != "before exact quoted text after" {
		t.Fatalf("Snippet = %q", snippet.Snippet)
	}
	if snippet.StartChar != 7 || snippet.EndChar != 24 {
		t.Fatalf("positions = %d..%d", snippet.StartChar, snippet.EndChar)
	}
}
