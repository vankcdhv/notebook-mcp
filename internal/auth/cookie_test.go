package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/store"
)

func TestImportCookieHeader(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "profile.json")
	s := store.New(profilePath)
	if err := ImportCookieHeader(s, "SID=abc; HSID=def"); err != nil {
		t.Fatal(err)
	}
	profile, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Cookies) != 2 {
		t.Fatalf("cookies = %d", len(profile.Cookies))
	}
	if profile.Cookies[0].Domain != ".google.com" {
		t.Fatalf("domain = %q", profile.Cookies[0].Domain)
	}
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatal(err)
	}
}

func TestImportCookieHeaderEmpty(t *testing.T) {
	s := store.New(filepath.Join(t.TempDir(), "profile.json"))
	if err := ImportCookieHeader(s, ""); err == nil {
		t.Fatal("expected error")
	}
}
