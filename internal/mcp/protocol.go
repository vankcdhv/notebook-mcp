package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const protocolVersion = "2024-11-05"

type Server struct {
	Name    string
	Version string
	Tools   []Tool
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]any         `json:"inputSchema"`
	Handler     func(context.Context, map[string]any) (any, error) `json:"-"`
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = writeMessage(writer, errorResponse(nil, -32700, "parse error"))
			continue
		}
		if req.ID == nil {
			_ = s.handleNotification(ctx, req)
			continue
		}
		if err := writeMessage(writer, s.handleRequest(ctx, req)); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func writeMessage(writer *bufio.Writer, resp response) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	return writer.Flush()
}

func (s *Server) handleNotification(ctx context.Context, req request) error {
	_ = ctx
	_ = req
	return nil
}

func (s *Server) handleRequest(ctx context.Context, req request) response {
	switch req.Method {
	case "initialize":
		return successResponse(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    s.Name,
				"version": s.Version,
			},
		})
	case "tools/list":
		tools := make([]Tool, 0, len(s.Tools))
		for _, tool := range s.Tools {
			tools = append(tools, Tool{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema})
		}
		return successResponse(req.ID, map[string]any{"tools": tools})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

type callParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) callTool(ctx context.Context, req request) response {
	var params callParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid tool call params")
	}
	for _, tool := range s.Tools {
		if tool.Name != params.Name {
			continue
		}
		result, err := tool.Handler(ctx, params.Arguments)
		if err != nil {
			return successResponse(req.ID, map[string]any{
				"isError": true,
				"content": []map[string]string{{"type": "text", "text": err.Error()}},
			})
		}
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return errorResponse(req.ID, -32603, "failed to encode tool result")
		}
		return successResponse(req.ID, map[string]any{
			"content": []map[string]string{{"type": "text", "text": string(payload)}},
		})
	}
	return errorResponse(req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
}

func successResponse(id any, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id any, code int, message string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: message}}
}
