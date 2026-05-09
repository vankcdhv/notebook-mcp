package mcp

func ObjectSchema(required []string, properties map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             required,
		"properties":           properties,
		"additionalProperties": false,
	}
}

func StringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func OptionalBoolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}
