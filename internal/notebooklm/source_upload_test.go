package notebooklm

import (
	"context"
	"os"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

type fakeUploadRPC struct {
	calls      []string
	uploadPath string
}

func (f *fakeUploadRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	f.calls = append(f.calls, method)
	return []any{[]any{"source-id", "Document"}}, nil
}

func (f *fakeUploadRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "", "", nil, nil
}

func (f *fakeUploadRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return nil, "", nil, nil
}

func (f *fakeUploadRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	f.uploadPath = filePath
	return nil
}

func TestAddFileRegistersThenUploadsFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "doc-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	rpcClient := &fakeUploadRPC{}
	client := New(rpcClient)

	source, err := client.AddFile(context.Background(), "nb-id", file.Name(), "text/plain")
	if err != nil {
		t.Fatalf("AddFile returned error: %v", err)
	}
	if source.ID != "source-id" {
		t.Fatalf("source id = %q, want source-id", source.ID)
	}
	if len(rpcClient.calls) != 1 || rpcClient.calls[0] != rpc.AddSourceFile {
		t.Fatalf("calls = %#v, want AddSourceFile", rpcClient.calls)
	}
	if rpcClient.uploadPath != file.Name() {
		t.Fatalf("upload path = %q, want %q", rpcClient.uploadPath, file.Name())
	}
}
