package notebooklm

import (
	"fmt"
	"strings"
	"time"
)

func parseNotebook(data any) Notebook {
	arr := asArray(data)
	title := strings.TrimSpace(strings.ReplaceAll(asString(at(arr, 0)), "thought\n", ""))
	createdAt := parseTimestamp(at(asArray(at(arr, 5)), 5))
	isOwner := true
	metadata := asArray(at(arr, 5))
	if len(metadata) > 1 {
		if shared, ok := metadata[1].(bool); ok {
			isOwner = !shared
		}
	}
	return Notebook{ID: asString(at(arr, 2)), Title: title, CreatedAt: createdAt, IsOwner: isOwner}
}

func parseSource(data any) Source {
	arr := asArray(data)
	sourceID := asString(at(arr, 0))
	if idArr := asArray(at(arr, 0)); len(idArr) > 0 {
		sourceID = asString(idArr[0])
	}
	metadata := asArray(at(arr, 2))
	status := 2
	statusData := asArray(at(arr, 3))
	if len(statusData) > 1 {
		if value := asInt(statusData[1]); value != 0 {
			status = value
		}
	}
	return Source{
		ID:        sourceID,
		Title:     asString(at(arr, 1)),
		URL:       extractSourceURL(metadata),
		TypeCode:  asInt(at(metadata, 4)),
		Status:    status,
		CreatedAt: parseTimestamp(at(metadata, 2)),
	}
}

func parseAddedSource(data any) Source {
	arr := unwrapSingleArrays(data)
	entry := asArray(arr)
	if len(entry) > 0 {
		if nested := asArray(entry[0]); len(nested) > 0 {
			entry = nested
		}
	}
	sourceID := asString(at(entry, 0))
	if idArr := asArray(at(entry, 0)); len(idArr) > 0 {
		sourceID = asString(idArr[0])
	}
	metadata := asArray(at(entry, 2))
	return Source{ID: sourceID, Title: asString(at(entry, 1)), URL: extractSourceURL(metadata), TypeCode: asInt(at(metadata, 4)), Status: 1}
}

func parseNote(data any) Note {
	arr := asArray(data)
	if len(arr) >= 3 && asInt(at(arr, 2)) == 2 {
		return Note{}
	}
	return Note{ID: asString(at(arr, 0)), Title: asString(at(arr, 4)), Content: asString(at(arr, 1))}
}

func parseResearchResult(data any) ResearchResult {
	arr := asArray(data)
	status := "in_progress"
	if asInt(at(arr, 0)) == 2 || asInt(at(arr, 0)) == 6 {
		status = "completed"
	}
	return ResearchResult{TaskID: asString(at(arr, 0)), Status: status, Title: asString(at(arr, 1)), URL: asString(at(arr, 2))}
}

func extractSourceURL(metadata []any) string {
	for _, idx := range []int{7, 5, 0} {
		value := asString(at(metadata, idx))
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value
		}
		arr := asArray(at(metadata, idx))
		for _, candidate := range arr {
			value = asString(candidate)
			if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
				return value
			}
		}
	}
	return ""
}

func extractFirstString(data any) string {
	if s, ok := data.(string); ok {
		return s
	}
	for _, item := range asArray(data) {
		if found := extractFirstString(item); found != "" {
			return found
		}
	}
	return ""
}

func unwrapSingleArrays(data any) any {
	for {
		arr := asArray(data)
		if len(arr) != 1 {
			return data
		}
		data = arr[0]
	}
}

func parseTimestamp(data any) *time.Time {
	arr := asArray(data)
	if len(arr) == 0 {
		return nil
	}
	seconds := int64(asFloat(arr[0]))
	if seconds <= 0 {
		return nil
	}
	t := time.Unix(seconds, 0).UTC()
	return &t
}

func asArray(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return nil
}

func at(arr []any, idx int) any {
	if idx < 0 || idx >= len(arr) {
		return nil
	}
	return arr[idx]
}

func asString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func asInt(v any) int {
	return int(asFloat(v))
}

func asFloat(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}
