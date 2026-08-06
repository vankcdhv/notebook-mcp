package notebooklm

import (
	"context"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

type recordingRPC struct {
	method    string
	params    []any
	allowNull bool
	result    any
}

func (r *recordingRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	r.method = method
	r.params = params
	r.allowNull = allowNull
	return r.result, nil
}

func (r *recordingRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "", "", nil, nil
}

func (r *recordingRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return nil, "", nil, nil
}

func (r *recordingRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestAddDriveBuildsDrivePayload(t *testing.T) {
	rpcClient := &recordingRPC{result: []any{[]any{"source-id", "Drive Doc"}}}
	client := New(rpcClient)

	_, err := client.AddDrive(context.Background(), "nb-id", "drive-id", "Drive Doc", "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if rpcClient.method != rpc.AddSource {
		t.Fatalf("method = %q, want %q", rpcClient.method, rpc.AddSource)
	}
	if !rpcClient.allowNull {
		t.Fatal("allowNull = false, want true")
	}
	sources := rpcClient.params[0].([]any)
	sourceData := sources[0].([]any)
	drive := sourceData[0].([]any)
	if drive[0] != "drive-id" || drive[1] != "application/pdf" || drive[3] != "Drive Doc" {
		t.Fatalf("drive payload = %#v", drive)
	}
}

func TestCreateNoteCreatesThenUpdatesContent(t *testing.T) {
	rpcClient := &recordingRPC{result: []any{[]any{"note-id", "", nil, nil, "New Note"}}}
	client := New(rpcClient)

	note, err := client.CreateNote(context.Background(), "nb-id", "Title", "Body")
	if err != nil {
		t.Fatal(err)
	}
	if note.ID != "note-id" || note.Title != "Title" || note.Content != "Body" {
		t.Fatalf("note = %+v", note)
	}
	if rpcClient.method != rpc.UpdateNote {
		t.Fatalf("last method = %q, want update note", rpcClient.method)
	}
}

func TestStartResearchRejectsDeepDrive(t *testing.T) {
	client := New(&recordingRPC{})

	_, err := client.StartResearch(context.Background(), "nb-id", "query", "drive", "deep")
	if err == nil {
		t.Fatal("StartResearch returned nil error")
	}
}

func TestImportResearchBuildsWebAndReportEntries(t *testing.T) {
	rpcClient := &recordingRPC{result: []any{[]any{"source-id", "Article"}}}
	client := New(rpcClient)

	_, err := client.ImportResearch(context.Background(), "nb-id", "task-id", []ResearchResult{
		{Title: "Article", URL: "https://example.com/a", Type: "web"},
		{Title: "Report", Content: "# Report", Type: "report"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rpcClient.method != rpc.ImportResearchMethod {
		t.Fatalf("method = %q, want import research", rpcClient.method)
	}
	entries := rpcClient.params[4].([]any)
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	web := entries[0].([]any)
	if web[10] != 2 {
		t.Fatalf("web entry kind = %v, want 2", web[10])
	}
	report := entries[1].([]any)
	if report[3] != 3 || report[10] != 3 {
		t.Fatalf("report entry = %#v", report)
	}
}
