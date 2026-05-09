package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vankcdhv/notebook-mcp/internal/auth"
)

type AuthProvider interface {
	Ensure(context.Context) (auth.Tokens, *http.Client, error)
}

type Client struct {
	Auth   AuthProvider
	ReqID  atomic.Int64
	BL     string
}

func NewClient(authProvider AuthProvider) *Client {
	return &Client{Auth: authProvider, BL: defaultBL()}
}

func (c *Client) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	return c.call(ctx, method, params, sourcePath, allowNull, false)
}

func (c *Client) call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool, retried bool) (any, error) {
	tokens, httpClient, err := c.Auth.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := EncodeRequest(method, params)
	if err != nil {
		return nil, err
	}
	body, err := BuildBody(encoded, tokens.CSRFToken)
	if err != nil {
		return nil, err
	}
	if sourcePath == "" {
		sourcePath = "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BuildURL(BatchExecuteURL, method, sourcePath, tokens.SessionID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if isAuthStatus(res.StatusCode) && !retried {
		return c.call(ctx, method, params, sourcePath, allowNull, true)
	}
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("NotebookLM rate limited RPC %s", method)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("NotebookLM RPC %s failed with HTTP %d", method, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 20*1024*1024))
	if err != nil {
		return nil, err
	}
	return DecodeResponse(string(data), method, allowNull)
}

func (c *Client) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []ChatReference, error) {
	return c.ask(ctx, notebookID, question, sourceIDs, conversationID, false)
}

func (c *Client) ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string, retried bool) (string, string, []ChatReference, error) {
	tokens, httpClient, err := c.Auth.Ensure(ctx)
	if err != nil {
		return "", "", nil, err
	}
	if conversationID == "" {
		conversationID = newConversationID()
	}
	sourcesArray := make([]any, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		sourcesArray = append(sourcesArray, []any{[]any{id}})
	}
	params := []any{
		sourcesArray,
		question,
		nil,
		[]any{2, nil, []any{1}, []any{1}},
		conversationID,
		nil,
		nil,
		notebookID,
		1,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", "", nil, err
	}
	freqJSON, err := json.Marshal([]any{nil, string(paramsJSON)})
	if err != nil {
		return "", "", nil, err
	}
	values := url.Values{}
	values.Set("f.req", string(freqJSON))
	if tokens.CSRFToken != "" {
		values.Set("at", tokens.CSRFToken)
	}
	query := url.Values{}
	query.Set("bl", c.BL)
	query.Set("hl", "en")
	query.Set("_reqid", fmt.Sprintf("%d", c.ReqID.Add(100000)))
	query.Set("rt", "c")
	if tokens.SessionID != "" {
		query.Set("f.sid", tokens.SessionID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, QueryURL+"?"+query.Encode(), bytes.NewBufferString(values.Encode()+"&"))
	if err != nil {
		return "", "", nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", "", nil, err
	}
	defer res.Body.Close()
	if isAuthStatus(res.StatusCode) && !retried {
		return c.ask(ctx, notebookID, question, sourceIDs, conversationID, true)
	}
	if res.StatusCode >= 400 {
		return "", "", nil, fmt.Errorf("NotebookLM chat failed with HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 20*1024*1024))
	if err != nil {
		return "", "", nil, err
	}
	answer, serverConversationID, refs, err := DecodeChatResponse(string(data))
	if serverConversationID != "" {
		conversationID = serverConversationID
	}
	return answer, conversationID, refs, err
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func defaultBL() string {
	if value := os.Getenv("NOTEBOOKLM_BL"); value != "" {
		return value
	}
	return "boq_labs-tailwind-ui_20250520.08_p0"
}

func newConversationID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}
