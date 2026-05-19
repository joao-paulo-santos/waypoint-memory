package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func toolError(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

func toolResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError("marshal result: %v", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

func requireInt(req mcp.CallToolRequest, key string) (int64, error) {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("%s is required", key)
	}
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

func requireString(req mcp.CallToolRequest, key string) (string, error) {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return "", fmt.Errorf("%s is required", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return s, nil
}

func getString(req mcp.CallToolRequest, key, def string) string {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func getInt(req mcp.CallToolRequest, key string, def int) int {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return def
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return def
	}
}

func getBool(req mcp.CallToolRequest, key string, def bool) bool {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

func getNullableString(req mcp.CallToolRequest, key string) *string {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

func getNullableInt(req mcp.CallToolRequest, key string) *int64 {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case int:
		i := int64(val)
		return &i
	case int64:
		return &val
	case float64:
		i := int64(val)
		return &i
	default:
		return nil
	}
}

func getNullableBool(req mcp.CallToolRequest, key string) *bool {
	v, ok := req.GetArguments()[key]
	if !ok || v == nil {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

func optString(ctx context.Context, args map[string]any, key string) *string {
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}
