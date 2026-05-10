package rpc

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/auth"
)

type retryAuthProvider struct {
	client *http.Client
}

func (p retryAuthProvider) Ensure(context.Context) (auth.Tokens, *http.Client, error) {
	return auth.Tokens{CSRFToken: "csrf", SessionID: "sid"}, p.client, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestCallRetriesTransientStatusThenSucceeds(t *testing.T) {
	attempts := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return responseWithStatus(http.StatusServiceUnavailable, "unavailable"), nil
		}
		return responseWithStatus(http.StatusOK, `)]}'
[["wrb.fr","wXbhsf","[[[\"Title\",null,\"nb-id\"]]]",null,null,null,"generic"]]`), nil
	})}
	client := NewClient(retryAuthProvider{client: httpClient})

	got, err := client.Call(context.Background(), ListNotebooks, []any{nil, 1, nil, []any{2}}, "/", false)
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	outer := got.([]any)
	first := outer[0].([]any)
	nb := first[0].([]any)
	if nb[2] != "nb-id" {
		t.Fatalf("notebook id = %v", nb[2])
	}
}

func TestCallDoesNotRetryNonRetryableStatus(t *testing.T) {
	attempts := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return responseWithStatus(http.StatusBadRequest, "bad request"), nil
	})}
	client := NewClient(retryAuthProvider{client: httpClient})

	_, err := client.Call(context.Background(), ListNotebooks, []any{nil, 1, nil, []any{2}}, "/", false)
	if err == nil {
		t.Fatal("Call returned nil error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func responseWithStatus(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
