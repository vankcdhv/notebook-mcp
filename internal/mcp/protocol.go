package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const protocolVersion = "2024-11-05"

type Server struct {
	Name    string
	Version string
	Tools   []Tool
}

type Tool struct {
	Name        string                                             `json:"name"`
	Description string                                             `json:"description"`
	InputSchema map[string]any                                     `json:"inputSchema"`
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
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	defer writer.Flush()
	useFraming := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		body, framed, err := readMessage(reader)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if framed {
			useFraming = true
		}

		var req request
		if err := json.Unmarshal(body, &req); err != nil {
			_ = writeMessage(writer, errorResponse(nil, -32700, "parse error"), useFraming)
			continue
		}
		if req.ID == nil {
			_ = s.handleNotification(ctx, req)
			continue
		}
		if err := writeMessage(writer, s.handleRequest(ctx, req), useFraming); err != nil {
			return err
		}
	}
}

func readMessage(reader *bufio.Reader) ([]byte, bool, error) {
	for {
		b, err := reader.Peek(1)
		if err != nil {
			return nil, false, err
		}
		switch b[0] {
		case '\n', '\r':
			if _, err := reader.ReadByte(); err != nil {
				return nil, false, err
			}
			continue
		case '{', '[':
			line, err := reader.ReadBytes('\n')
			if err != nil && len(line) == 0 {
				return nil, false, err
			}
			body := bytes.TrimRight(line, "\r\n")
			return body, false, nil
		default:
			body, err := readContentLengthMessage(reader)
			return body, true, err
		}
	}
}

func readContentLengthMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("malformed header: %s", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			length, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || length < 0 {
				return nil, fmt.Errorf("invalid Content-Length: %s", strings.TrimSpace(value))
			}
			contentLength = length
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeMessage(writer *bufio.Writer, resp response, framed bool) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if framed {
		if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
			return err
		}
		if _, err := writer.Write(payload); err != nil {
			return err
		}
		return writer.Flush()
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
