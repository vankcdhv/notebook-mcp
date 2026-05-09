package auth

import (
	"errors"
	"regexp"
)

type Tokens struct {
	CSRFToken string
	SessionID string
}

var (
	csrfPatterns = []*regexp.Regexp{
		regexp.MustCompile(`"SNlM0e"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`SNlM0e["']?\s*[,=:]\s*["']([^"']+)`),
	}
	sessionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`"FdrFJe"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`FdrFJe["']?\s*[,=:]\s*["']([^"']+)`),
	}
)

func ExtractTokens(html string) (Tokens, error) {
	csrf := firstMatch(html, csrfPatterns)
	session := firstMatch(html, sessionPatterns)
	if csrf == "" || session == "" {
		return Tokens{}, errors.New("could not extract NotebookLM CSRF/session tokens")
	}
	return Tokens{CSRFToken: csrf, SessionID: session}, nil
}

func firstMatch(s string, patterns []*regexp.Regexp) string {
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(s)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}
