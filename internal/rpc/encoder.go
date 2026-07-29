package rpc

import (
	"encoding/json"
	"net/url"
)

func EncodeRequest(method string, params []any) ([]any, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	inner := []any{method, string(paramsJSON), nil, "generic"}
	return []any{[]any{inner}}, nil
}

func BuildBody(request []any, csrfToken string) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("f.req", string(payload))
	if csrfToken != "" {
		values.Set("at", csrfToken)
	}
	return values.Encode() + "&", nil
}

func BuildURL(baseURL, method, sourcePath, sessionID, buildLabel string) string {
	values := url.Values{}
	values.Set("rpcids", method)
	values.Set("source-path", sourcePath)
	values.Set("hl", "en")
	values.Set("rt", "c")
	if sessionID != "" {
		values.Set("f.sid", sessionID)
	}
	if buildLabel != "" {
		values.Set("bl", buildLabel)
	}
	return baseURL + "?" + values.Encode()
}
