package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestServerServeReadsContentLengthFramedRequests(t *testing.T) {
	server := &Server{Name: "test-server", Version: "test-version"}
	input := framedJSON(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	var output bytes.Buffer

	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	resp := readFramedResponse(t, output.String())
	result := resp["result"].(map[string]any)
	if result["protocolVersion"] != protocolVersion {
		t.Fatalf("protocolVersion = %v, want %v", result["protocolVersion"], protocolVersion)
	}
}

func TestServerServeReadsBackToBackContentLengthFramedRequests(t *testing.T) {
	server := &Server{Name: "test-server", Version: "test-version"}
	input := framedJSON(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`) + framedJSON(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	var output bytes.Buffer

	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	first, rest := readOneFramedResponse(t, output.String())
	second, rest := readOneFramedResponse(t, rest)
	if strings.TrimSpace(rest) != "" {
		t.Fatalf("unexpected trailing output: %q", rest)
	}
	if first["id"].(float64) != 1 {
		t.Fatalf("first id = %v, want 1", first["id"])
	}
	if second["id"].(float64) != 2 {
		t.Fatalf("second id = %v, want 2", second["id"])
	}
}

func TestServerServeRejectsMalformedFrame(t *testing.T) {
	server := &Server{Name: "test-server", Version: "test-version"}
	var output bytes.Buffer

	err := server.Serve(context.Background(), strings.NewReader("Content-Type: application/json\r\n\r\n{}"), &output)
	if err == nil {
		t.Fatal("Serve returned nil error for missing Content-Length")
	}
}

func framedJSON(payload string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)
}

func readFramedResponse(t *testing.T, data string) map[string]any {
	t.Helper()
	resp, rest := readOneFramedResponse(t, data)
	if strings.TrimSpace(rest) != "" {
		t.Fatalf("unexpected trailing output: %q", rest)
	}
	return resp
}

func readOneFramedResponse(t *testing.T, data string) (map[string]any, string) {
	t.Helper()
	reader := strings.NewReader(data)
	var length int
	for {
		line, err := readHeaderLine(reader)
		if err != nil {
			t.Fatalf("reading header: %v", err)
		}
		if line == "\r\n" {
			break
		}
		if _, err := fmt.Sscanf(line, "Content-Length: %d\r\n", &length); err != nil && strings.HasPrefix(line, "Content-Length:") {
			t.Fatalf("bad content-length header %q: %v", line, err)
		}
	}
	if length <= 0 {
		t.Fatal("missing Content-Length in response")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		t.Fatalf("reading body: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(body), err)
	}
	remaining, _ := io.ReadAll(reader)
	return resp, string(remaining)
}

func readHeaderLine(reader *strings.Reader) (string, error) {
	var builder strings.Builder
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return "", err
		}
		builder.WriteRune(r)
		if r == '\n' {
			return builder.String(), nil
		}
	}
}
