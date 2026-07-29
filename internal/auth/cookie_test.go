package auth

import (
	"net/url"
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

func TestInstallCookiesKeepsHostScopedCookies(t *testing.T) {
	m, err := NewManager(filepath.Join(t.TempDir(), "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.installCookies(&store.Profile{Cookies: []store.Cookie{
		{Name: "SID", Value: "shared", Domain: ".google.com", Path: "/"},
		{Name: "OSID", Value: "legacy-host", Domain: "notebooklm.google.com", Path: "/"},
		{Name: "OSID", Value: "new-host", Domain: "notebook.google.com", Path: "/"},
	}})

	for _, tc := range []struct{ rawURL, wantOSID string }{
		{"https://notebooklm.google.com/", "legacy-host"},
		{"https://notebook.google.com/", "new-host"},
	} {
		u, err := url.Parse(tc.rawURL)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, cookie := range m.HTTPClient.Jar.Cookies(u) {
			got[cookie.Name] = cookie.Value
		}
		if got["SID"] != "shared" {
			t.Fatalf("%s SID = %q", tc.rawURL, got["SID"])
		}
		if got["OSID"] != tc.wantOSID {
			t.Fatalf("%s OSID = %q, want %q", tc.rawURL, got["OSID"], tc.wantOSID)
		}
	}
}
