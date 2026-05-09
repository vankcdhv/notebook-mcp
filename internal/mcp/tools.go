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
	AddText(context.Context, string, string, string) (notebooklm.Source, error)
	DeleteSource(context.Context, string, string) (bool, error)
	Ask(context.Context, string, string, []string, string) (notebooklm.AskResult, error)
	KnowledgeSearch(context.Context, string, string, []string, int, int) (notebooklm.KnowledgeSearchResult, error)
}

func NotebookLMTools(client NotebookLMClient) []Tool {
	return []Tool{
		{Name: "notebook_list", Description: "List NotebookLM notebooks", InputSchema: ObjectSchema(nil, map[string]any{"include_shared": OptionalBoolProp("Include shared notebooks")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebooks, err := client.ListNotebooks(ctx); if err != nil { return nil, err }
			if include, ok := args["include_shared"].(bool); ok && !include { filtered := notebooks[:0]; for _, nb := range notebooks { if nb.IsOwner { filtered = append(filtered, nb) } }; notebooks = filtered }
			return map[string]any{"count": len(notebooks), "notebooks": notebooks}, nil
		}},
		{Name: "notebook_get", Description: "Get a notebook with summary and sources", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }
			notebook, err := client.GetNotebook(ctx, notebookID); if err != nil { return nil, err }
			sources, _ := client.ListSources(ctx, notebookID)
			summary, _ := client.GetSummary(ctx, notebookID)
			return map[string]any{"notebook": notebook, "summary": summary, "sources": sources}, nil
		}},
		{Name: "notebook_create", Description: "Create a NotebookLM notebook", InputSchema: ObjectSchema([]string{"title"}, map[string]any{"title": StringProp("Notebook title")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			title, err := requireString(args, "title"); if err != nil { return nil, err }
			notebook, err := client.CreateNotebook(ctx, title); return map[string]any{"notebook": notebook}, err
		}},
		{Name: "source_list", Description: "List sources in a notebook", InputSchema: ObjectSchema([]string{"notebook_id"}, map[string]any{"notebook_id": StringProp("Notebook ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }
			sources, err := client.ListSources(ctx, notebookID); return map[string]any{"count": len(sources), "sources": sources}, err
		}},
		{Name: "source_add", Description: "Add a URL or text source to a notebook", InputSchema: ObjectSchema([]string{"notebook_id", "type", "content"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "type": StringProp("Source type: url or text"), "content": StringProp("URL or text content"), "title": StringProp("Title for text source")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }
			sourceType, err := requireString(args, "type"); if err != nil { return nil, err }
			content, err := requireString(args, "content"); if err != nil { return nil, err }
			var source notebooklm.Source
			switch sourceType { case "url": source, err = client.AddURL(ctx, notebookID, content); case "text": title, _ := optionalString(args, "title", "Pasted Text"); source, err = client.AddText(ctx, notebookID, title, content); default: return nil, fmt.Errorf("type must be url or text") }
			return map[string]any{"source": source}, err
		}},
		{Name: "source_delete", Description: "Delete a source from a notebook", InputSchema: ObjectSchema([]string{"notebook_id", "source_id"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "source_id": StringProp("Source ID")}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }; sourceID, err := requireString(args, "source_id"); if err != nil { return nil, err }; ok, err := client.DeleteSource(ctx, notebookID, sourceID); return map[string]bool{"success": ok}, err
		}},
		{Name: "ask", Description: "Ask NotebookLM a blocking chat question", InputSchema: ObjectSchema([]string{"notebook_id", "question"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "question": StringProp("Question"), "conversation_id": StringProp("Conversation ID for follow-up"), "source_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional source IDs to scope the question"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }; question, err := requireString(args, "question"); if err != nil { return nil, err }; conversationID, _ := optionalString(args, "conversation_id", ""); return client.Ask(ctx, notebookID, question, stringSlice(args["source_ids"]), conversationID)
		}},
		{Name: "knowledge_search", Description: "Search NotebookLM knowledge and return citation-backed snippets", InputSchema: ObjectSchema([]string{"notebook_id", "query"}, map[string]any{"notebook_id": StringProp("Notebook ID"), "query": StringProp("Search query"), "context_chars": map[string]any{"type": "number", "description": "Context characters around exact quote"}, "max_results": map[string]any{"type": "number", "description": "Maximum snippets"}, "source_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional source IDs to scope search"}}), Handler: func(ctx context.Context, args map[string]any) (any, error) {
			notebookID, err := requireString(args, "notebook_id"); if err != nil { return nil, err }; query, err := requireString(args, "query"); if err != nil { return nil, err }; return client.KnowledgeSearch(ctx, notebookID, query, stringSlice(args["source_ids"]), intArg(args, "context_chars", 300), intArg(args, "max_results", 10))
		}},
	}
}

func requireString(args map[string]any, key string) (string, error) { value, ok := args[key].(string); if !ok || value == "" { return "", fmt.Errorf("missing string argument %s", key) }; return value, nil }
func optionalString(args map[string]any, key string, fallback string) (string, bool) { value, ok := args[key].(string); if !ok || value == "" { return fallback, false }; return value, true }
func stringSlice(value any) []string { raw, ok := value.([]any); if !ok { return nil }; out := make([]string, 0, len(raw)); for _, item := range raw { if s, ok := item.(string); ok && s != "" { out = append(out, s) } }; return out }
func intArg(args map[string]any, key string, fallback int) int { switch v := args[key].(type) { case float64: return int(v); case int: return v; default: return fallback } }
