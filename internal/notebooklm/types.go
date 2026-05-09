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
