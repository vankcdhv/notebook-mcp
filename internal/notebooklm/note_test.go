package notebooklm

import (
	"context"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

// fakeNoteRPC replays the payload shape NotebookLM returns for GetNotes:
// [[[id, note], ...], timestamp], where each note body is [id, content, meta, nil, title].
type fakeNoteRPC struct{}

func (f *fakeNoteRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	note := []any{"note-1", "Note body", []any{float64(1), "1094750087393"}, nil, "Note title"}
	return []any{
		[]any{[]any{"note-1", note}},
		[]any{float64(1785344557), float64(802908000)},
	}, nil
}

func (f *fakeNoteRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "", "", nil, nil
}

func (f *fakeNoteRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return nil, "", nil, nil
}

func (f *fakeNoteRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestListNotesReadsNestedPayload(t *testing.T) {
	notes, err := New(&fakeNoteRPC{}).ListNotes(context.Background(), "nb")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %d, want 1", len(notes))
	}
	if notes[0].ID != "note-1" || notes[0].Title != "Note title" || notes[0].Content != "Note body" {
		t.Fatalf("note = %+v", notes[0])
	}
}

func TestGetNoteFindsNote(t *testing.T) {
	note, err := New(&fakeNoteRPC{}).GetNote(context.Background(), "nb", "note-1")
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Note title" {
		t.Fatalf("note = %+v", note)
	}
}
