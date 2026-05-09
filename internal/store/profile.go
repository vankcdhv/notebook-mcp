package store

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	DirMode  os.FileMode = 0o700
	FileMode os.FileMode = 0o600
)

type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain,omitempty"`
	Path     string    `json:"path,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"http_only,omitempty"`
}

type Profile struct {
	Cookies   []Cookie  `json:"cookies"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	Path string
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", ".notebooklm-mcp", "profile.json")
	}
	return filepath.Join(home, ".notebooklm-mcp", "profile.json")
}

func New(path string) *Store {
	if path == "" {
		path = DefaultPath()
	}
	return &Store{Path: path}
}

func (s *Store) Load() (*Profile, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}
	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	if len(profile.Cookies) == 0 {
		return nil, errors.New("profile has no cookies")
	}
	return &profile, nil
}

func (s *Store) Save(profile *Profile) error {
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), DirMode); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(s.Path), DirMode); err != nil && !isWindows() {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.Path, data, FileMode); err != nil {
		return err
	}
	if !isWindows() {
		return os.Chmod(s.Path, FileMode)
	}
	return nil
}

func (p *Profile) HTTPCookies() []*http.Cookie {
	cookies := make([]*http.Cookie, 0, len(p.Cookies))
	for _, cookie := range p.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  cookie.Expires,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HTTPOnly,
		})
	}
	return cookies
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
