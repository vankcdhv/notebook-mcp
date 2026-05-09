package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ChatReference struct {
	SourceID  string `json:"source_id"`
	CitedText string `json:"cited_text,omitempty"`
	StartChar int    `json:"start_char,omitempty"`
	EndChar   int    `json:"end_char,omitempty"`
	ChunkID   string `json:"chunk_id,omitempty"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$|^source-[A-Za-z0-9_-]+$`)

func DecodeResponse(text, method string, allowNull bool) (any, error) {
	text = strings.TrimSpace(strings.TrimPrefix(text, ")]}'"))
	if text == "" {
		if allowNull { return nil, nil }
		return nil, errors.New("empty RPC response")
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || isLengthLine(line) { continue }
		var outer any
		if err := json.Unmarshal([]byte(line), &outer); err != nil { continue }
		payload, found, err := findWRBPayload(outer, method)
		if err != nil { return nil, err }
		if !found { continue }
		if payload == "" || payload == "null" {
			if allowNull { return nil, nil }
			return nil, fmt.Errorf("RPC %s returned null", method)
		}
		var decoded any
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil { return nil, fmt.Errorf("decode RPC %s payload: %w", method, err) }
		return decoded, nil
	}
	if allowNull { return nil, nil }
	return nil, fmt.Errorf("RPC %s payload not found", method)
}

func DecodeChatResponse(text string) (string, string, []ChatReference, error) {
	text = strings.TrimSpace(strings.TrimPrefix(text, ")]}'"))
	lines := strings.Split(text, "\n")
	bestMarked := ""
	bestUnmarked := ""
	conversationID := ""
	var bestMarkedRefs []ChatReference
	var bestUnmarkedRefs []ChatReference
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" { continue }
		if isLengthLine(line) && i+1 < len(lines) { i++; line = strings.TrimSpace(lines[i]) }
		text, marked, convID, refs := extractChatChunk(line)
		if convID != "" { conversationID = convID }
		if text == "" { continue }
		if marked && len(text) > len(bestMarked) { bestMarked = text; bestMarkedRefs = refs }
		if !marked && len(text) > len(bestUnmarked) { bestUnmarked = text; bestUnmarkedRefs = refs }
	}
	if bestMarked != "" { return bestMarked, conversationID, bestMarkedRefs, nil }
	if bestUnmarked != "" { return bestUnmarked, conversationID, bestUnmarkedRefs, nil }
	return "", conversationID, nil, errors.New("no chat answer found")
}

func extractChatChunk(line string) (string, bool, string, []ChatReference) {
	var outer any
	if err := json.Unmarshal([]byte(line), &outer); err != nil { return "", false, "", nil }
	payload, found, _ := findWRBPayload(outer, "")
	if !found || payload == "" { return "", false, "", nil }
	var inner any
	if err := json.Unmarshal([]byte(payload), &inner); err != nil { return "", false, "", nil }
	arr, ok := inner.([]any)
	if !ok || len(arr) == 0 { return "", false, "", nil }
	first, ok := arr[0].([]any)
	if !ok || len(first) == 0 { return "", false, "", nil }
	answer, _ := first[0].(string)
	marked := false
	if typeInfo := asArray(at(first, 4)); len(typeInfo) > 0 {
		if n, ok := typeInfo[len(typeInfo)-1].(float64); ok && n == 1 { marked = true }
	}
	conversationID := ""
	if conv := asArray(at(first, 2)); len(conv) > 0 { conversationID, _ = conv[0].(string) }
	return answer, marked, conversationID, parseCitations(first)
}

func parseCitations(first []any) []ChatReference {
	typeInfo := asArray(at(first, 4))
	citations := asArray(at(typeInfo, 3))
	refs := make([]ChatReference, 0, len(citations))
	for _, citation := range citations {
		if ref, ok := parseCitation(citation); ok { refs = append(refs, ref) }
	}
	return refs
}

func parseCitation(citation any) (ChatReference, bool) {
	cite := asArray(citation)
	if len(cite) < 2 { return ChatReference{}, false }
	inner := asArray(cite[1])
	if len(inner) == 0 { return ChatReference{}, false }
	sourceID := extractUUID(at(inner, 5), 10)
	if sourceID == "" { return ChatReference{}, false }
	ref := ChatReference{SourceID: sourceID}
	if chunk := asArray(at(cite, 0)); len(chunk) > 0 { ref.ChunkID, _ = chunk[0].(string) }
	ref.CitedText, ref.StartChar, ref.EndChar = extractTextPassages(inner)
	return ref, true
}

func extractTextPassages(inner []any) (string, int, int) {
	passages := asArray(at(inner, 4))
	texts := []string{}
	start, end := 0, 0
	for _, wrapper := range passages {
		wrap := asArray(wrapper)
		if len(wrap) == 0 { continue }
		passage := asArray(wrap[0])
		if len(passage) < 3 { continue }
		if start == 0 { start = asInt(passage[0]) }
		end = asInt(passage[1])
		collectTexts(at(passage, 2), &texts, 10)
	}
	return strings.Join(texts, " "), start, end
}

func collectTexts(v any, texts *[]string, depth int) {
	if depth <= 0 { return }
	arr := asArray(v)
	if arr == nil { if s, ok := v.(string); ok && strings.TrimSpace(s) != "" { *texts = append(*texts, strings.TrimSpace(s)) }; return }
	for _, item := range arr { collectTexts(item, texts, depth-1) }
}

func extractUUID(v any, depth int) string {
	if depth <= 0 || v == nil { return "" }
	if s, ok := v.(string); ok && uuidPattern.MatchString(s) { return s }
	for _, item := range asArray(v) { if found := extractUUID(item, depth-1); found != "" { return found } }
	return ""
}

func findWRBPayload(v any, method string) (string, bool, error) {
	arr, ok := v.([]any)
	if !ok { return "", false, nil }
	for _, item := range arr {
		inner, ok := item.([]any)
		if !ok || len(inner) < 3 { if payload, found, err := findWRBPayload(item, method); found || err != nil { return payload, found, err }; continue }
		tag, _ := inner[0].(string)
		if tag != "wrb.fr" { if payload, found, err := findWRBPayload(item, method); found || err != nil { return payload, found, err }; continue }
		if method != "" { got, _ := inner[1].(string); if got != method { continue } }
		payload, _ := inner[2].(string)
		return payload, true, nil
	}
	return "", false, nil
}

func asArray(v any) []any { if arr, ok := v.([]any); ok { return arr }; return nil }
func at(arr []any, idx int) any { if idx < 0 || idx >= len(arr) { return nil }; return arr[idx] }
func asInt(v any) int { if n, ok := v.(float64); ok { return int(n) }; if n, ok := v.(int); ok { return n }; return 0 }
func isLengthLine(s string) bool { for _, r := range s { if r < '0' || r > '9' { return false } }; return s != "" }
