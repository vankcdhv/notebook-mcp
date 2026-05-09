package mcp

func ObjectSchema(required []string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func StringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func OptionalBoolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}
