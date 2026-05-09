package notebooklm

import (
	"context"
	"fmt"
	"strings"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

type RPC interface {
	Call(context.Context, string, []any, string, bool) (any, error)
	Ask(context.Context, string, string, []string, string) (string, string, []rpc.ChatReference, error)
}

type Client struct { RPC RPC }
func New(rpcClient RPC) *Client { return &Client{RPC: rpcClient} }

func (c *Client) ListNotebooks(ctx context.Context) ([]Notebook, error) {
	result, err := c.RPC.Call(ctx, rpc.ListNotebooks, []any{nil, 1, nil, []any{2}}, "/", false)
	if err != nil { return nil, err }
	raw := asArray(result)
	if len(raw) > 0 { if nested := asArray(raw[0]); len(nested) > 0 { raw = nested } }
	notebooks := make([]Notebook, 0, len(raw))
	for _, item := range raw { notebook := parseNotebook(item); if notebook.ID != "" { notebooks = append(notebooks, notebook) } }
	return notebooks, nil
}

func (c *Client) CreateNotebook(ctx context.Context, title string) (Notebook, error) {
	result, err := c.RPC.Call(ctx, rpc.CreateNotebook, []any{title, nil, nil, []any{2}, []any{1}}, "/", false)
	if err != nil { return Notebook{}, err }
	return parseNotebook(result), nil
}

func (c *Client) GetNotebook(ctx context.Context, notebookID string) (Notebook, error) {
	result, err := c.RPC.Call(ctx, rpc.GetNotebook, []any{notebookID, nil, []any{2}, nil, 0}, "/notebook/"+notebookID, false)
	if err != nil { return Notebook{}, err }
	raw := asArray(result)
	if len(raw) == 0 { return Notebook{}, fmt.Errorf("notebook %s not found", notebookID) }
	return parseNotebook(raw[0]), nil
}

func (c *Client) GetSummary(ctx context.Context, notebookID string) (string, error) {
	result, err := c.RPC.Call(ctx, rpc.Summarize, []any{notebookID, []any{2}}, "/notebook/"+notebookID, false)
	if err != nil { return "", err }
	outer := asArray(result)
	if len(outer) == 0 { return "", nil }
	first := asArray(outer[0])
	if len(first) == 0 { return "", nil }
	summary := asArray(first[0])
	if len(summary) == 0 { return "", nil }
	return asString(summary[0]), nil
}

func (c *Client) ListSources(ctx context.Context, notebookID string) ([]Source, error) {
	result, err := c.RPC.Call(ctx, rpc.GetNotebook, []any{notebookID, nil, []any{2}, nil, 0}, "/notebook/"+notebookID, false)
	if err != nil { return nil, err }
	raw := asArray(result)
	if len(raw) == 0 { return []Source{}, nil }
	nbInfo := asArray(raw[0])
	sourcesRaw := asArray(at(nbInfo, 1))
	sources := make([]Source, 0, len(sourcesRaw))
	for _, item := range sourcesRaw { source := parseSource(item); if source.ID != "" { sources = append(sources, source) } }
	return sources, nil
}

func (c *Client) AddURL(ctx context.Context, notebookID, url string) (Source, error) {
	params := []any{[]any{[]any{nil, nil, []any{url}, nil, nil, nil, nil, nil}}, notebookID, []any{2}, nil, nil}
	result, err := c.RPC.Call(ctx, rpc.AddSource, params, "/notebook/"+notebookID, false)
	if err != nil { return Source{}, err }
	return parseAddedSource(result), nil
}

func (c *Client) AddText(ctx context.Context, notebookID, title, text string) (Source, error) {
	params := []any{[]any{[]any{nil, []any{title, text}, nil, nil, nil, nil, nil, nil}}, notebookID, []any{2}, nil, nil}
	result, err := c.RPC.Call(ctx, rpc.AddSource, params, "/notebook/"+notebookID, false)
	if err != nil { return Source{}, err }
	return parseAddedSource(result), nil
}

func (c *Client) DeleteSource(ctx context.Context, notebookID, sourceID string) (bool, error) {
	_, err := c.RPC.Call(ctx, rpc.DeleteSource, []any{[]any{[]any{[]any{sourceID}}}}, "/notebook/"+notebookID, true)
	return err == nil, err
}

func (c *Client) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (AskResult, error) {
	if sourceIDs == nil {
		sources, err := c.ListSources(ctx, notebookID)
		if err != nil { return AskResult{}, err }
		for _, source := range sources { sourceIDs = append(sourceIDs, source.ID) }
	}
	answer, convID, refs, err := c.RPC.Ask(ctx, notebookID, question, sourceIDs, conversationID)
	return AskResult{Answer: answer, ConversationID: convID, References: convertRefs(refs)}, err
}

func (c *Client) AskNotebook(ctx context.Context, notebookID, question string) (AskResult, error) {
	return c.Ask(ctx, notebookID, question, nil, "")
}

func (c *Client) AskSource(ctx context.Context, notebookID, sourceID, question string) (AskResult, error) {
	result, err := c.Ask(ctx, notebookID, question, []string{sourceID}, "")
	result.Warning = "source scoping depends on NotebookLM enforcing supplied source IDs"
	return result, err
}

func (c *Client) GetSourceFulltext(ctx context.Context, notebookID, sourceID string) (SourceFulltext, error) {
	result, err := c.RPC.Call(ctx, rpc.GetSource, []any{[]any{sourceID}, []any{2}, []any{2}}, "/notebook/"+notebookID, true)
	if err != nil { return SourceFulltext{}, err }
	raw := asArray(result)
	if len(raw) == 0 { return SourceFulltext{}, fmt.Errorf("source %s not found", sourceID) }
	entry := asArray(raw[0])
	metadata := asArray(at(entry, 2))
	blocks := asArray(at(asArray(at(raw, 3)), 0))
	texts := []string{}
	extractAllText(blocks, &texts, 100)
	content := strings.Join(texts, "\n")
	return SourceFulltext{SourceID: sourceID, Title: asString(at(entry, 1)), Content: content, TypeCode: asInt(at(metadata, 4)), URL: extractSourceURL(metadata), CharCount: len(content)}, nil
}

func (c *Client) KnowledgeSearch(ctx context.Context, notebookID, query string, sourceIDs []string, contextChars int, maxResults int) (KnowledgeSearchResult, error) {
	askResult, err := c.Ask(ctx, notebookID, query, sourceIDs, "")
	if err != nil { return KnowledgeSearchResult{}, err }
	if contextChars < 0 { contextChars = 0 }
	if maxResults <= 0 { maxResults = 10 }
	cache := map[string]SourceFulltext{}
	result := KnowledgeSearchResult{Answer: askResult.Answer, ConversationID: askResult.ConversationID, Snippets: []KnowledgeSnippet{}}
	for _, ref := range askResult.References {
		if len(result.Snippets) >= maxResults { break }
		fulltext, ok := cache[ref.SourceID]
		if !ok { fulltext, err = c.GetSourceFulltext(ctx, notebookID, ref.SourceID); if err != nil { continue }; cache[ref.SourceID] = fulltext }
		start, end := resolveOffsets(ref, fulltext.Content)
		if start < 0 || end <= start { continue }
		snippetStart := max(0, start-contextChars)
		snippetEnd := min(len(fulltext.Content), end+contextChars)
		result.Snippets = append(result.Snippets, KnowledgeSnippet{SourceID: ref.SourceID, SourceTitle: fulltext.Title, ExactQuote: ref.CitedText, Snippet: fulltext.Content[snippetStart:snippetEnd], StartChar: start, EndChar: end})
	}
	result.Count = len(result.Snippets)
	return result, nil
}

func resolveOffsets(ref ChatReference, content string) (int, int) {
	if ref.StartChar > 0 && ref.EndChar > ref.StartChar && ref.EndChar <= len(content) { return ref.StartChar, ref.EndChar }
	if ref.CitedText != "" { if idx := strings.Index(content, ref.CitedText); idx >= 0 { return idx, idx + len(ref.CitedText) } }
	return -1, -1
}

func convertRefs(refs []rpc.ChatReference) []ChatReference {
	out := make([]ChatReference, 0, len(refs))
	for _, ref := range refs { out = append(out, ChatReference{SourceID: ref.SourceID, CitedText: ref.CitedText, StartChar: ref.StartChar, EndChar: ref.EndChar, ChunkID: ref.ChunkID}) }
	return out
}

func extractAllText(data any, texts *[]string, depth int) {
	if depth <= 0 { return }
	if s, ok := data.(string); ok && s != "" { *texts = append(*texts, s); return }
	for _, item := range asArray(data) { extractAllText(item, texts, depth-1) }
}

func min(a, b int) int { if a < b { return a }; return b }
func max(a, b int) int { if a > b { return a }; return b }
