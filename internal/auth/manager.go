package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vankcdhv/notebook-mcp/internal/store"
	"golang.org/x/net/publicsuffix"
)

const homeURL = "https://notebooklm.google.com/"

type Manager struct {
	Store      *store.Store
	HTTPClient *http.Client
}

func NewManager(profilePath string) (*Manager, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	return &Manager{
		Store: store.New(profilePath),
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 120 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if isAllowedHost(req.URL.Hostname()) {
					return nil
				}
				return fmt.Errorf("refusing auth redirect to %s", req.URL.Hostname())
			},
		},
	}, nil
}

func (m *Manager) Ensure(ctx context.Context) (Tokens, *http.Client, error) {
	profile, err := m.Store.Load()
	if err != nil || len(profile.Cookies) == 0 {
		if err := m.Login(ctx); err != nil {
			return Tokens{}, nil, err
		}
		profile, err = m.Store.Load()
		if err != nil {
			return Tokens{}, nil, err
		}
	}
	m.installCookies(profile)
	tokens, err := m.RefreshTokens(ctx)
	if err != nil {
		return Tokens{}, nil, fmt.Errorf("auth refresh failed; run `notebooklm-mcp login` again: %w", err)
	}
	return tokens, m.HTTPClient, nil
}

func (m *Manager) Login(ctx context.Context) error {
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("start Playwright: %w", err)
	}
	defer pw.Stop()

	browserProfileDir, err := browserProfileDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(browserProfileDir, store.DirMode); err != nil {
		return err
	}

	browserCtx, err := pw.Chromium.LaunchPersistentContext(browserProfileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
		Channel:  playwright.String("chrome"),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--password-store=basic",
		},
		IgnoreDefaultArgs: []string{"--enable-automation"},
	})
	if err != nil {
		browserCtx, err = pw.Chromium.LaunchPersistentContext(browserProfileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless: playwright.Bool(false),
			Args: []string{
				"--disable-blink-features=AutomationControlled",
				"--password-store=basic",
			},
			IgnoreDefaultArgs: []string{"--enable-automation"},
		})
		if err != nil {
			return fmt.Errorf("launch Playwright browser: %w", err)
		}
	}
	defer browserCtx.Close()

	page, err := browserCtx.NewPage()
	if err != nil {
		return err
	}
	if _, err := page.Goto(homeURL); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Complete Google login in the browser, wait for the NotebookLM homepage, then press ENTER here.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	for _, target := range []string{"https://accounts.google.com/", homeURL} {
		_, _ = page.Goto(target, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateCommit})
	}
	if err := waitForNotebookLM(ctx, page); err != nil {
		return err
	}

	browserCookies, err := browserCtx.Cookies()
	if err != nil {
		return err
	}
	if len(browserCookies) == 0 {
		return errors.New("no cookies captured after login")
	}

	profile := &store.Profile{UpdatedAt: time.Now().UTC()}
	for _, cookie := range browserCookies {
		if !isAllowedCookieDomain(cookie.Domain) {
			continue
		}
		expires := time.Time{}
		if cookie.Expires > 0 {
			expires = time.Unix(int64(cookie.Expires), 0).UTC()
		}
		profile.Cookies = append(profile.Cookies, store.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  expires,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HttpOnly,
		})
	}
	if len(profile.Cookies) == 0 {
		return errors.New("no Google/NotebookLM cookies captured after login")
	}
	if err := m.Store.Save(profile); err != nil {
		return err
	}
	m.installCookies(profile)
	if _, err := m.RefreshTokens(ctx); err != nil {
		return fmt.Errorf("saved browser cookies but NotebookLM token verification failed: %w", err)
	}
	return nil
}

func waitForNotebookLM(ctx context.Context, page playwright.Page) error {
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		currentURL := page.URL()
		if isNotebookLMURL(currentURL) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for NotebookLM login")
}

func (m *Manager) RefreshTokens(ctx context.Context) (Tokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, homeURL, nil)
	if err != nil {
		return Tokens{}, err
	}
	res, err := m.HTTPClient.Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return Tokens{}, fmt.Errorf("NotebookLM auth rejected with HTTP %d", res.StatusCode)
	}
	if res.StatusCode >= 400 {
		return Tokens{}, fmt.Errorf("NotebookLM homepage returned HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024))
	if err != nil {
		return Tokens{}, err
	}
	return ExtractTokens(string(body))
}

func (m *Manager) installCookies(profile *store.Profile) {
	u, _ := url.Parse(homeURL)
	m.HTTPClient.Jar.SetCookies(u, profile.HTTPCookies())
}

func browserProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot resolve user home directory")
	}
	return filepath.Join(home, ".notebooklm-mcp", "browser-profile"), nil
}

func isNotebookLMURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return isAllowedHost(u.Hostname()) && u.Hostname() == "notebooklm.google.com"
}

func isAllowedHost(host string) bool {
	return host == "notebooklm.google.com" || host == "accounts.google.com" || host == "myaccount.google.com" || host == "ssl.gstatic.com" || host == "www.gstatic.com"
}

func ImportCookieHeader(profileStore *store.Store, header string) error {
	cookies := parseCookieHeader(header)
	if len(cookies) == 0 {
		return errors.New("cookie header contains no cookies")
	}
	profile := &store.Profile{UpdatedAt: time.Now().UTC()}
	for _, cookie := range cookies {
		profile.Cookies = append(profile.Cookies, store.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   ".google.com",
			Path:     "/",
			Secure:   true,
			HTTPOnly: true,
		})
	}
	return profileStore.Save(profile)
}

func parseCookieHeader(header string) []*http.Cookie {
	req := &http.Request{Header: http.Header{"Cookie": []string{header}}}
	return req.Cookies()
}

func isAllowedCookieDomain(domain string) bool {
	if strings.HasSuffix(domain, ".google.com") || domain == "google.com" {
		return true
	}
	if strings.HasSuffix(domain, ".googleusercontent.com") {
		return true
	}
	return domain == "notebooklm.google.com" || domain == ".notebooklm.google.com"
}

func HasProfile(path string) bool {
	if path == "" {
		path = store.DefaultPath()
	}
	_, err := os.Stat(path)
	return err == nil
}
