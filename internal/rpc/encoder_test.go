package rpc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeRequest(t *testing.T) {
	encoded, err := EncodeRequest("wXbhsf", []any{nil, 1, nil, []any{2}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	want := `[[["wXbhsf","[null,1,null,[2]]",null,"generic"]]]`
	if string(data) != want {
		t.Fatalf("got %s want %s", data, want)
	}
}

func TestBuildBody(t *testing.T) {
	encoded, _ := EncodeRequest("wXbhsf", []any{nil})
	body, err := BuildBody(encoded, "csrf-token")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(body, "&") {
		t.Fatalf("body should end with &: %q", body)
	}
	if !strings.Contains(body, "at=csrf-token") {
		t.Fatalf("missing csrf in body: %q", body)
	}
	if !strings.Contains(body, "f.req=") {
		t.Fatalf("missing f.req in body: %q", body)
	}
}

func TestBuildURL(t *testing.T) {
	got := BuildURL("https://example.com/x", "wXbhsf", "/notebook/abc", "session-id", "")
	if !strings.Contains(got, "rpcids=wXbhsf") {
		t.Fatalf("missing rpcids: %s", got)
	}
	if !strings.Contains(got, "source-path=%2Fnotebook%2Fabc") {
		t.Fatalf("missing source-path: %s", got)
	}
	if !strings.Contains(got, "f.sid=session-id") {
		t.Fatalf("missing session id: %s", got)
	}
	if !strings.Contains(got, "rt=c") {
		t.Fatalf("missing rt=c: %s", got)
	}
}
