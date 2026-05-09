package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveProfilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	s := New(path)
	err := s.Save(&Profile{Cookies: []Cookie{{Name: "SID", Value: "secret", Domain: ".google.com", Path: "/"}}})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Cookies) != 1 || loaded.Cookies[0].Name != "SID" {
		t.Fatalf("unexpected profile: %+v", loaded)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != FileMode {
		t.Fatalf("mode = %o want %o", info.Mode().Perm(), FileMode)
	}
}
