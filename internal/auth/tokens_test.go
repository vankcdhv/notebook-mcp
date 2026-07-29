package auth

import "testing"

func TestExtractTokens(t *testing.T) {
	tokens, err := ExtractTokens(`{"SNlM0e":"csrf-token","FdrFJe":"session-id"}`)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.CSRFToken != "csrf-token" {
		t.Fatalf("CSRFToken = %q", tokens.CSRFToken)
	}
	if tokens.SessionID != "session-id" {
		t.Fatalf("SessionID = %q", tokens.SessionID)
	}
}

func TestExtractTokensMissing(t *testing.T) {
	_, err := ExtractTokens(`{}`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractTokensBuildLabel(t *testing.T) {
	tokens, err := ExtractTokens(`{"SNlM0e":"csrf-token","FdrFJe":"session-id","cfb2h":"boq_labs-tailwind-frontend_20260727.10_p0"}`)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.BuildLabel != "boq_labs-tailwind-frontend_20260727.10_p0" {
		t.Fatalf("BuildLabel = %q", tokens.BuildLabel)
	}
}
