package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/notebooklm"
)

type fakeNotebookLMClient struct {
	notebookErr             error
	sourcesErr              error
	summaryErr              error
	lastMethod              string
	lastImportedResearchLen int
	lastImportedResearch    []notebooklm.ResearchResult
}

func (f fakeNotebookLMClient) ListNotebooks(context.Context) ([]notebooklm.Notebook, error) {
	return nil, nil
}
func (f fakeNotebookLMClient) GetNotebook(context.Context, string) (notebooklm.Notebook, error) {
	return notebooklm.Notebook{ID: "nb-id", Title: "Notebook"}, f.notebookErr
}
func (f fakeNotebookLMClient) CreateNotebook(context.Context, string) (notebooklm.Notebook, error) {
	return notebooklm.Notebook{}, nil
}
func (f fakeNotebookLMClient) GetSummary(context.Context, string) (string, error) {
	return "summary", f.summaryErr
}
func (f fakeNotebookLMClient) ListSources(context.Context, string) ([]notebooklm.Source, error) {
	return []notebooklm.Source{{ID: "src-id", Title: "Source"}}, f.sourcesErr
}
func (f fakeNotebookLMClient) AddURL(context.Context, string, string) (notebooklm.Source, error) {
	return notebooklm.Source{ID: "url"}, nil
}
func (f fakeNotebookLMClient) AddText(context.Context, string, string, string) (notebooklm.Source, error) {
	return notebooklm.Source{ID: "text"}, nil
}
func (f fakeNotebookLMClient) AddYouTube(context.Context, string, string) (notebooklm.Source, error) {
	return notebooklm.Source{ID: "youtube"}, nil
}
func (f fakeNotebookLMClient) AddDrive(context.Context, string, string, string, string) (notebooklm.Source, error) {
	return notebooklm.Source{ID: "drive"}, nil
}
func (f fakeNotebookLMClient) AddFile(context.Context, string, string, string) (notebooklm.Source, error) {
	return notebooklm.Source{ID: "file"}, nil
}
func (f fakeNotebookLMClient) DeleteSource(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f fakeNotebookLMClient) ListNotes(context.Context, string) ([]notebooklm.Note, error) {
	return []notebooklm.Note{{ID: "note-id", Title: "Note"}}, nil
}
func (f fakeNotebookLMClient) GetNote(context.Context, string, string) (notebooklm.Note, error) {
	return notebooklm.Note{ID: "note-id", Title: "Note"}, nil
}
func (f fakeNotebookLMClient) CreateNote(context.Context, string, string, string) (notebooklm.Note, error) {
	return notebooklm.Note{ID: "created", Title: "Created"}, nil
}
func (f fakeNotebookLMClient) UpdateNote(context.Context, string, string, string, string) (notebooklm.Note, error) {
	return notebooklm.Note{ID: "updated", Title: "Updated"}, nil
}
func (f fakeNotebookLMClient) DeleteNote(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f fakeNotebookLMClient) StartResearch(context.Context, string, string, string, string) (notebooklm.ResearchResult, error) {
	return notebooklm.ResearchResult{TaskID: "task-id", Status: "started"}, nil
}
func (f fakeNotebookLMClient) PollResearch(context.Context, string) ([]notebooklm.ResearchResult, error) {
	return []notebooklm.ResearchResult{{TaskID: "task-id", Status: "completed"}}, nil
}
func (f *fakeNotebookLMClient) ImportResearch(ctx context.Context, notebookID, taskID string, sources []notebooklm.ResearchResult) ([]notebooklm.Source, error) {
	f.lastImportedResearchLen = len(sources)
	f.lastImportedResearch = sources
	return []notebooklm.Source{{ID: "imported"}}, nil
}
func (f fakeNotebookLMClient) Ask(context.Context, string, string, []string, string) (notebooklm.AskResult, error) {
	return notebooklm.AskResult{}, nil
}
func (f fakeNotebookLMClient) AskStream(context.Context, string, string, []string, string) (notebooklm.AskStreamResult, error) {
	return notebooklm.AskStreamResult{Answer: "answer", Chunks: []notebooklm.AskChunk{{Seq: 1, Text: "answer", IsFinal: true}}}, nil
}
func (f fakeNotebookLMClient) KnowledgeSearch(context.Context, string, string, []string, int, int) (notebooklm.KnowledgeSearchResult, error) {
	return notebooklm.KnowledgeSearchResult{}, nil
}

func TestNotebookGetReturnsWarningsForPartialFailures(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{sourcesErr: errors.New("sources failed"), summaryErr: errors.New("summary failed")})
	tool := findTool(t, tools, "notebook_get")

	result, err := tool.Handler(context.Background(), map[string]any{"notebook_id": "nb-id"})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	payload := result.(map[string]any)
	warnings := payload["warnings"].([]string)
	if len(warnings) != 2 {
		t.Fatalf("warnings len = %d, want 2", len(warnings))
	}
	if payload["notebook"].(notebooklm.Notebook).ID != "nb-id" {
		t.Fatalf("missing notebook payload")
	}
}

func TestNotebookGetStrictModeFailsOnPartialFailures(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{summaryErr: errors.New("summary failed")})
	tool := findTool(t, tools, "notebook_get")

	_, err := tool.Handler(context.Background(), map[string]any{"notebook_id": "nb-id", "fail_on_partial": true})
	if err == nil {
		t.Fatal("Handler returned nil error")
	}
}

func TestSourceAddDispatchesExpandedSourceTypes(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{})
	tool := findTool(t, tools, "source_add")
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "youtube", args: map[string]any{"notebook_id": "nb-id", "type": "youtube", "url": "https://youtu.be/video"}, want: "youtube"},
		{name: "drive", args: map[string]any{"notebook_id": "nb-id", "type": "drive", "file_id": "drive-id", "title": "Drive Doc"}, want: "drive"},
		{name: "file", args: map[string]any{"notebook_id": "nb-id", "type": "file", "file_path": "/tmp/doc.pdf"}, want: "file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Handler(context.Background(), tc.args)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}
			source := result.(map[string]any)["source"].(notebooklm.Source)
			if source.ID != tc.want {
				t.Fatalf("source id = %q, want %q", source.ID, tc.want)
			}
		})
	}
}

func TestNoteToolsAreRegistered(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{})
	for _, name := range []string{"note_list", "note_get", "note_create", "note_update", "note_delete"} {
		findTool(t, tools, name)
	}
}

func TestResearchToolsAreRegistered(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{})
	for _, name := range []string{"research_start", "research_poll", "research_import"} {
		findTool(t, tools, name)
	}
}

func TestAskStreamToolIsRegistered(t *testing.T) {
	tools := NotebookLMTools(&fakeNotebookLMClient{})
	findTool(t, tools, "ask_stream")
}

func TestResearchImportParsesSourcesArgument(t *testing.T) {
	client := &fakeNotebookLMClient{}
	tools := NotebookLMTools(client)
	tool := findTool(t, tools, "research_import")

	_, err := tool.Handler(context.Background(), map[string]any{
		"notebook_id": "nb-id",
		"task_id":     "task-id",
		"sources": []any{
			map[string]any{"title": "Article", "url": "https://example.com/a", "type": "web"},
			map[string]any{"title": "Report", "content": "# Report", "type": "report"},
		},
	})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if client.lastImportedResearchLen != 2 {
		t.Fatalf("imported research len = %d, want 2", client.lastImportedResearchLen)
	}
}

func TestResearchImportKeepsReportContentSeparateFromURL(t *testing.T) {
	client := &fakeNotebookLMClient{}
	tools := NotebookLMTools(client)
	tool := findTool(t, tools, "research_import")

	_, err := tool.Handler(context.Background(), map[string]any{
		"notebook_id": "nb-id",
		"task_id":     "task-id",
		"sources": []any{
			map[string]any{"title": "Report", "content": "# Report", "type": "report"},
		},
	})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	imported := client.lastImportedResearch[0]
	if imported.Content != "# Report" {
		t.Fatalf("content = %q", imported.Content)
	}
	if imported.URL != "" {
		t.Fatalf("report content leaked into url: %q", imported.URL)
	}
}

func TestResearchImportRequiresSources(t *testing.T) {
	client := &fakeNotebookLMClient{}
	tools := NotebookLMTools(client)
	tool := findTool(t, tools, "research_import")

	_, err := tool.Handler(context.Background(), map[string]any{
		"notebook_id": "nb-id",
		"task_id":     "task-id",
	})
	if err == nil {
		t.Fatal("Handler reported success without anything to import")
	}
}

func findTool(t *testing.T, tools []Tool, name string) Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return Tool{}
}
