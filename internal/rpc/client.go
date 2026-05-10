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
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vankcdhv/notebook-mcp/internal/auth"
	"github.com/vankcdhv/notebook-mcp/internal/obs"
)

type AuthProvider interface {
	Ensure(context.Context) (auth.Tokens, *http.Client, error)
}

type Client struct {
	Auth       AuthProvider
	ReqID      atomic.Int64
	BL         string
	RetryDelay time.Duration
}

func NewClient(authProvider AuthProvider) *Client {
	return &Client{Auth: authProvider, BL: defaultBL(), RetryDelay: 100 * time.Millisecond}
}

func (c *Client) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	return c.call(ctx, method, params, sourcePath, allowNull, false)
}

func (c *Client) call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool, retried bool) (any, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		result, status, err := c.callOnce(ctx, method, params, sourcePath, allowNull)
		if err == nil {
			return result, nil
		}
		if isAuthStatus(status) && !retried {
			return c.call(ctx, method, params, sourcePath, allowNull, true)
		}
		lastErr = err
		if !isRetryableStatus(status) || attempt == 2 {
			break
		}
		if err := sleepForRetry(ctx, c.retryDelay(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, redactError(lastErr)
}

func (c *Client) callOnce(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, int, error) {
	tokens, httpClient, err := c.Auth.Ensure(ctx)
	if err != nil {
		return nil, 0, err
	}
	encoded, err := EncodeRequest(method, params)
	if err != nil {
		return nil, 0, err
	}
	body, err := BuildBody(encoded, tokens.CSRFToken)
	if err != nil {
		return nil, 0, err
	}
	if sourcePath == "" {
		sourcePath = "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BuildURL(BatchExecuteURL, method, sourcePath, tokens.SessionID), strings.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, res.StatusCode, fmt.Errorf("NotebookLM rate limited RPC %s", method)
	}
	if res.StatusCode >= 400 {
		return nil, res.StatusCode, fmt.Errorf("NotebookLM RPC %s failed with HTTP %d", method, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 20*1024*1024))
	if err != nil {
		return nil, res.StatusCode, err
	}
	result, err := DecodeResponse(string(data), method, allowNull)
	return result, res.StatusCode, err
}

func (c *Client) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []ChatReference, error) {
	return c.ask(ctx, notebookID, question, sourceIDs, conversationID, false)
}

func (c *Client) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]ChatChunk, string, []ChatReference, error) {
	text, err := c.askResponse(ctx, notebookID, question, sourceIDs, conversationID, false)
	if err != nil {
		return nil, "", nil, err
	}
	return DecodeChatChunks(text)
}

func (c *Client) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	tokens, httpClient, err := c.Auth.Ensure(ctx)
	if err != nil {
		return err
	}
	startBody, err := json.Marshal(map[string]string{
		"PROJECT_ID":  notebookID,
		"SOURCE_NAME": filepath.Base(filePath),
		"SOURCE_ID":   sourceID,
	})
	if err != nil {
		return err
	}
	startReq, err := http.NewRequestWithContext(ctx, http.MethodPost, UploadURL+"?authuser=0", bytes.NewReader(startBody))
	if err != nil {
		return err
	}
	setUploadHeaders(startReq, tokens)
	startReq.Header.Set("x-goog-upload-command", "start")
	startReq.Header.Set("x-goog-upload-header-content-length", fmt.Sprintf("%d", info.Size()))
	startReq.Header.Set("x-goog-upload-protocol", "resumable")
	startRes, err := httpClient.Do(startReq)
	if err != nil {
		return err
	}
	defer startRes.Body.Close()
	if startRes.StatusCode >= 400 {
		return fmt.Errorf("NotebookLM upload start failed with HTTP %d", startRes.StatusCode)
	}
	uploadURL := startRes.Header.Get("x-goog-upload-url")
	if uploadURL == "" {
		return fmt.Errorf("NotebookLM upload start did not return upload URL")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, file)
	if err != nil {
		return err
	}
	setUploadHeaders(uploadReq, tokens)
	uploadReq.Header.Set("x-goog-upload-command", "upload, finalize")
	uploadReq.Header.Set("x-goog-upload-offset", "0")
	if mimeType != "" {
		uploadReq.Header.Set("Content-Type", mimeType)
	}
	uploadRes, err := httpClient.Do(uploadReq)
	if err != nil {
		return err
	}
	defer uploadRes.Body.Close()
	if uploadRes.StatusCode >= 400 {
		return fmt.Errorf("NotebookLM upload failed with HTTP %d", uploadRes.StatusCode)
	}
	return nil
}

func setUploadHeaders(req *http.Request, tokens auth.Tokens) {
	_ = tokens
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Origin", "https://notebooklm.google.com")
	req.Header.Set("Referer", "https://notebooklm.google.com/")
	req.Header.Set("x-goog-authuser", "0")
}

func (c *Client) askResponse(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string, retried bool) (string, error) {
	tokens, httpClient, err := c.Auth.Ensure(ctx)
	if err != nil {
		return "", err
	}
	if conversationID == "" {
		conversationID = newConversationID()
	}
	sourcesArray := make([]any, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		sourcesArray = append(sourcesArray, []any{[]any{id}})
	}
	params := []any{sourcesArray, question, nil, []any{2, nil, []any{1}, []any{1}}, conversationID, nil, nil, notebookID, 1}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	freqJSON, err := json.Marshal([]any{nil, string(paramsJSON)})
	if err != nil {
		return "", err
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
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if isAuthStatus(res.StatusCode) && !retried {
		return c.askResponse(ctx, notebookID, question, sourceIDs, conversationID, true)
	}
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("NotebookLM chat failed with HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 20*1024*1024))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) askRaw(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string, retried bool) (string, string, []ChatReference, error) {
	text, err := c.askResponse(ctx, notebookID, question, sourceIDs, conversationID, retried)
	if err != nil {
		return "", conversationID, nil, err
	}
	answer, serverConversationID, refs, err := DecodeChatResponse(text)
	if serverConversationID != "" {
		conversationID = serverConversationID
	}
	return answer, conversationID, refs, err
}

func (c *Client) ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string, retried bool) (string, string, []ChatReference, error) {
	return c.askRaw(ctx, notebookID, question, sourceIDs, conversationID, retried)
}

func redactError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", obs.Redact(err.Error()))
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func (c *Client) retryDelay(attempt int) time.Duration {
	if c.RetryDelay <= 0 {
		return 0
	}
	return c.RetryDelay << attempt
}

func sleepForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
