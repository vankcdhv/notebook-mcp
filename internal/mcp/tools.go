package mcp

import (
	"context"
	"fmt"

	"github.com/vankcdhv/notebook-mcp/internal/notebooklm"
)

type NotebookLMClient interface {
	ListNotebooks(context.Context) ([]notebooklm.Notebook, error)
	GetNotebook(context.Context, string) (notebooklm.Notebook, error)
	CreateNotebook(context.Context, string) (notebooklm.Notebook, error)
	GetSummary(context.Context, string) (string, error)
	ListSources(context.Context, string) ([]notebooklm.Source, error)
	AddURL(context.Context, string, string) (notebooklm.Source, error)
	AddYouTube(context.Context, string, string) (notebooklm.Source, error)
	AddText(context.Context, string, string, string) (notebooklm.Source, error)
	AddDrive(context.Context, string, string, string, string) (notebooklm.Source, error)
	AddFile(context.Context, string, string, string) (notebooklm.Source, error)
	DeleteSource(context.Context, string, string) (bool, error)
	ListNotes(context.Context, string) ([]notebooklm.Note, error)
	GetNote(context.Context, string, string) (notebooklm.Note, error)
	CreateNote(context.Context, string, string, string) (notebooklm.Note, error)
	UpdateNote(context.Context, string, string, string, string) (notebooklm.Note, error)
	DeleteNote(context.Context, string, string) (bool, error)
	StartResearch(context.Context, string, string, string, string) (notebooklm.ResearchResult, error)
	PollResearch(context.Context, string) ([]notebooklm.ResearchResult, error)
	ImportResearch(context.Context, string, string, []notebooklm.ResearchResult) ([]notebooklm.Source, error)
	Ask(context.Context, string, string, []string, string) (notebooklm.AskResult, error)
	AskStream(context.Context, string, string, []string, string) (notebooklm.AskStreamResult, error)
	KnowledgeSearch(context.Context, string, string, []string, int, int) (notebooklm.KnowledgeSearchResult, error)
}

func NotebookLMTools(client NotebookLMClient) []Tool {
	return []Tool{
		{Name: "notebook_list", Description: "List NotebookLM notebooks", InputSchema: ObjectSchema(nil, map[string]any{"include_shared": OptionalBoolProp("Include shared notebooks")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebooks, err := client.ListNotebooks(ctx)
			if err != nil {
				return nil, err
			}
			if include, ok := args["include_shared"].(bool); ok && !include {
				filtered := notebooks[:0]
				for _, nb := range notebooks {
					if nb.IsOwner {
						filtered = append(filtered, nb)
					}
				}
				notebooks = filtered
			}
			return map[string]any{"count": len(notebooks), "notebooks": notebooks}, nil
		}},
		{Name: "notebook_get", Description: "Get a notebook with summary and sources", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "fail_on_partial": OptionalBoolProp("Fail if summary or sources cannot be loaded")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			notebook, err := client.GetNotebook(ctx, notebookID)
			if err != nil {
				return nil, err
			}
			failOnPartial, _ := args["fail_on_partial"].(bool)
			warnings := []string{}
			sources, err := client.ListSources(ctx, notebookID)
			if err != nil {
				if failOnPartial {
					return nil, err
				}
				warnings = append(warnings, fmt.Sprintf("sources: %v", err))
			}
			summary, err := client.GetSummary(ctx, notebookID)
			if err != nil {
				if failOnPartial {
					return nil, err
				}
				warnings = append(warnings, fmt.Sprintf("summary: %v", err))
			}
			return map[string]any{"notebook": notebook, "summary": summary, "sources": sources, "warnings": warnings}, nil
		}},
		{Name: "notebook_create", Description: "Create a NotebookLM notebook", InputSchema: ObjectSchema([]string{"title"}, map[string]any{"title": StringProp("Notebook title")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			title, err := requireString(args, "title")
			if err != nil {
				return nil, err
			}
			notebook, err := client.CreateNotebook(ctx, title)
			return map[string]any{"notebook": notebook}, err
		}},
		{Name: "source_list", Description: "List sources in a notebook", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			sources, err := client.ListSources(ctx, notebookID)
			return map[string]any{"count": len(sources), "sources": sources}, err
		}},
		{Name: "source_add", Description: "Add a source to a notebook", InputSchema: ObjectSchema([]string{"notebook_id", "type"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "type": StringProp("Source type: url, youtube, text, drive, or file"), "url": StringProp("URL or YouTube URL"), "content": StringProp("Text source content"), "title": StringProp("Source title"), "file_id": StringProp("Google Drive file ID"), "file_path": StringProp("Local file path"), "mime_type": StringProp("MIME type")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			sourceType, err := requireString(args, "type")
			if err != nil {
				return nil, err
			}
			var source notebooklm.Source
			switch sourceType {
			case "url":
				url, err := requireString(args, "url")
				if err != nil {
					return nil, err
				}
				source, err = client.AddURL(ctx, notebookID, url)
			case "youtube":
				url, err := requireString(args, "url")
				if err != nil {
					return nil, err
				}
				source, err = client.AddYouTube(ctx, notebookID, url)
			case "text":
				content, err := requireString(args, "content")
				if err != nil {
					return nil, err
				}
				title, _ := optionalString(args, "title", "Pasted Text")
				source, err = client.AddText(ctx, notebookID, title, content)
			case "drive":
				fileID, err := requireString(args, "file_id")
				if err != nil {
					return nil, err
				}
				title, err := requireString(args, "title")
				if err != nil {
					return nil, err
				}
				mimeType, _ := optionalString(args, "mime_type", "application/vnd.google-apps.document")
				source, err = client.AddDrive(ctx, notebookID, fileID, title, mimeType)
			case "file":
				filePath, err := requireString(args, "file_path")
				if err != nil {
					return nil, err
				}
				mimeType, _ := optionalString(args, "mime_type", "")
				source, err = client.AddFile(ctx, notebookID, filePath, mimeType)
			default:
				return nil, fmt.Errorf("type must be url, youtube, text, drive, or file")
			}
			return map[string]any{"source": source}, err
		}},
		{Name: "source_delete", Description: "Delete a source from a notebook", InputSchema: ObjectSchema([]string{"notebook_id", "source_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "source_id": StringProp("Source ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			sourceID, err := requireString(args, "source_id")
			if err != nil {
				return nil, err
			}
			ok, err := client.DeleteSource(ctx, notebookID, sourceID)
			return map[string]bool{"success": ok}, err
		}},
		{Name: "note_list", Description: "List notes in a notebook", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			notes, err := client.ListNotes(ctx, notebookID)
			return map[string]any{"count": len(notes), "notes": notes}, err
		}},
		{Name: "note_get", Description: "Get a note", InputSchema: ObjectSchema([]string{"notebook_id", "note_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "note_id": StringProp("Note ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			noteID, err := requireString(args, "note_id")
			if err != nil {
				return nil, err
			}
			note, err := client.GetNote(ctx, notebookID, noteID)
			return map[string]any{"note": note}, err
		}},
		{Name: "note_create", Description: "Create a note", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "title": StringProp("Note title"), "content": StringProp("Note content")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			title, _ := optionalString(args, "title", "New Note")
			content, _ := optionalString(args, "content", "")
			note, err := client.CreateNote(ctx, notebookID, title, content)
			return map[string]any{"note": note}, err
		}},
		{Name: "note_update", Description: "Update a note", InputSchema: ObjectSchema([]string{"notebook_id", "note_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "note_id": StringProp("Note ID"), "title": StringProp("Note title"), "content": StringProp("Note content")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			noteID, err := requireString(args, "note_id")
			if err != nil {
				return nil, err
			}
			title, _ := optionalString(args, "title", "")
			content, _ := optionalString(args, "content", "")
			note, err := client.UpdateNote(ctx, notebookID, noteID, title, content)
			return map[string]any{"note": note}, err
		}},
		{Name: "note_delete", Description: "Delete a note", InputSchema: ObjectSchema([]string{"notebook_id", "note_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "note_id": StringProp("Note ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			noteID, err := requireString(args, "note_id")
			if err != nil {
				return nil, err
			}
			ok, err := client.DeleteNote(ctx, notebookID, noteID)
			return map[string]bool{"success": ok}, err
		}},
		{Name: "research_start", Description: "Start NotebookLM research", InputSchema: ObjectSchema([]string{"notebook_id", "query"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "query": StringProp("Research query"), "source": StringProp("web or drive"), "mode": StringProp("fast or deep")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			query, err := requireString(args, "query")
			if err != nil {
				return nil, err
			}
			source, _ := optionalString(args, "source", "web")
			mode, _ := optionalString(args, "mode", "fast")
			result, err := client.StartResearch(ctx, notebookID, query, source, mode)
			return map[string]any{"research": result}, err
		}},
		{Name: "research_poll", Description: "Poll NotebookLM research", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			results, err := client.PollResearch(ctx, notebookID)
			return map[string]any{"count": len(results), "results": results}, err
		}},
		{Name: "research_import", Description: "Import NotebookLM research results", InputSchema: ObjectSchema([]string{"notebook_id", "task_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "task_id": StringProp("Research task ID"), "sources": map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Research results to import"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			taskID, err := requireString(args, "task_id")
			if err != nil {
				return nil, err
			}
			results := researchResultsArg(args["sources"])
			sources, err := client.ImportResearch(ctx, notebookID, taskID, results)
			return map[string]any{"count": len(sources), "sources": sources}, err
		}},
		{Name: "ask", Description: "Ask NotebookLM a blocking chat question", InputSchema: ObjectSchema([]string{"notebook_id", "question"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "question": StringProp("Question"), "conversation_id": StringProp("Conversation ID for follow-up"), "source_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional source IDs to scope the question"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			question, err := requireString(args, "question")
			if err != nil {
				return nil, err
			}
			conversationID, _ := optionalString(args, "conversation_id", "")
			return client.Ask(ctx, notebookID, question, stringSlice(args["source_ids"]), conversationID)
		}},
		{Name: "ask_stream", Description: "Ask NotebookLM and return structured chunks", InputSchema: ObjectSchema([]string{"notebook_id", "question"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "question": StringProp("Question"), "conversation_id": StringProp("Conversation ID for follow-up"), "source_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional source IDs to scope the question"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			question, err := requireString(args, "question")
			if err != nil {
				return nil, err
			}
			conversationID, _ := optionalString(args, "conversation_id", "")
			return client.AskStream(ctx, notebookID, question, stringSlice(args["source_ids"]), conversationID)
		}},
		{Name: "knowledge_search", Description: "Search NotebookLM knowledge and return citation-backed snippets", InputSchema: ObjectSchema([]string{"notebook_id", "query"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "query": StringProp("Search query"), "context_chars": map[string]any{"type": "number", "description": "Context characters around exact quote"}, "max_results": map[string]any{"type": "number", "description": "Maximum snippets"}, "source_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional source IDs to scope search"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id")
			if err != nil {
				return nil, err
			}
			query, err := requireString(args, "query")
			if err != nil {
				return nil, err
			}
			return client.KnowledgeSearch(ctx, notebookID, query, stringSlice(args["source_ids"]), intArg(args, "context_chars", 300), intArg(args, "max_results", 10))
		}},
	}
}

func requireString(args map[string]any, key string) (string, error) {
	value, ok := args[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("missing string argument %s", key)
	}
	return value, nil
}
func optionalString(args map[string]any, key string, fallback string) (string, bool) {
	value, ok := args[key].(string)
	if !ok || value == "" {
		return fallback, false
	}
	return value, true
}
func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
func intArg(args map[string]any, key string, fallback int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return fallback
	}
}

func researchResultsArg(value any) []notebooklm.ResearchResult {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]notebooklm.ResearchResult, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result := notebooklm.ResearchResult{}
		if s, ok := obj["task_id"].(string); ok {
			result.TaskID = s
		}
		if s, ok := obj["status"].(string); ok {
			result.Status = s
		}
		if s, ok := obj["title"].(string); ok {
			result.Title = s
		}
		if s, ok := obj["url"].(string); ok {
			result.URL = s
		}
		if s, ok := obj["type"].(string); ok {
			result.Type = s
		}
		if s, ok := obj["content"].(string); ok && result.URL == "" {
			result.URL = s
		}
		out = append(out, result)
	}
	return out
}
