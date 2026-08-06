package notebooklm

import "time"

type Notebook struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	IsOwner   bool       `json:"is_owner"`
}

type Source struct {
	ID        string     `json:"id"`
	Title     string     `json:"title,omitempty"`
	URL       string     `json:"url,omitempty"`
	TypeCode  int        `json:"type_code,omitempty"`
	Status    int        `json:"status"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type Note struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

// ResearchResult is one importable item a research task produced: either a web
// page it consulted, or the report deep research generated. Report items carry
// their markdown in Content and have no URL.
type ResearchResult struct {
	TaskID  string `json:"task_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

type ChatReference struct {
	SourceID  string `json:"source_id"`
	CitedText string `json:"cited_text,omitempty"`
	StartChar int    `json:"start_char,omitempty"`
	EndChar   int    `json:"end_char,omitempty"`
	ChunkID   string `json:"chunk_id,omitempty"`
}

type AskResult struct {
	Answer         string          `json:"answer"`
	ConversationID string          `json:"conversation_id,omitempty"`
	References     []ChatReference `json:"references,omitempty"`
	Warning        string          `json:"warning,omitempty"`
}

type AskChunk struct {
	Seq     int    `json:"seq"`
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

type AskStreamResult struct {
	Answer         string          `json:"answer"`
	ConversationID string          `json:"conversation_id,omitempty"`
	References     []ChatReference `json:"references,omitempty"`
	Chunks         []AskChunk      `json:"chunks"`
}

type SourceFulltext struct {
	SourceID  string `json:"source_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	TypeCode  int    `json:"type_code,omitempty"`
	URL       string `json:"url,omitempty"`
	CharCount int    `json:"char_count"`
}

type KnowledgeSnippet struct {
	SourceID    string `json:"source_id"`
	SourceTitle string `json:"source_title"`
	ExactQuote  string `json:"exact_quote"`
	Snippet     string `json:"snippet"`
	StartChar   int    `json:"start_char"`
	EndChar     int    `json:"end_char"`
}

type KnowledgeSearchResult struct {
	Answer         string             `json:"answer"`
	ConversationID string             `json:"conversation_id,omitempty"`
	Count          int                `json:"count"`
	Snippets       []KnowledgeSnippet `json:"snippets"`
}
