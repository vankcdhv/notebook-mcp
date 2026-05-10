package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vankcdhv/notebook-mcp/internal/auth"
	"github.com/vankcdhv/notebook-mcp/internal/mcp"
	"github.com/vankcdhv/notebook-mcp/internal/notebooklm"
	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

const version = "0.2.0"

const usage = `notebooklm-mcp — MCP stdio server for Google NotebookLM

Usage:
  notebooklm-mcp                  Run the MCP stdio server (default; reads JSON-RPC from stdin).
  notebooklm-mcp setup [--yes]    First-time setup: install Playwright browser if needed, then login.
  notebooklm-mcp install-browsers Install Playwright browser only.
  notebooklm-mcp login            Interactive Google login via Playwright.
  notebooklm-mcp login --cookie "SID=...; HSID=...; ..."
                                  Import a Cookie header instead of using the browser flow.
  notebooklm-mcp version          Print the version.
  notebooklm-mcp --help           Show this help.

Profile lives at ~/.notebooklm-mcp/ (override with NOTEBOOKLM_MCP_PROFILE).
`

func runSetup(ctx context.Context, authManager *auth.Manager, yes bool) error {
	if err := ensurePlaywrightBrowsers(yes); err != nil {
		return err
	}
	if err := authManager.Login(ctx); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		executable = "notebooklm-mcp"
	}
	fmt.Fprintln(os.Stderr, "Setup complete.")
	fmt.Fprintf(os.Stderr, "MCP command: %s\n", executable)
	return nil
}

func ensurePlaywrightBrowsers(yes bool) error {
	pw, err := playwright.Run()
	if err == nil {
		_ = pw.Stop()
		return nil
	}
	if !yes && !confirm("Playwright browsers are not installed or not runnable. Install now? [y/N] ") {
		return fmt.Errorf("Playwright browsers are required; run `notebooklm-mcp install-browsers`")
	}
	return playwright.Install()
}

func confirm(prompt string) bool {
	fmt.Fprint(os.Stderr, prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

func hasArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func main() {
	log.SetOutput(os.Stderr)
	ctx := context.Background()

	profilePath := os.Getenv("NOTEBOOKLM_MCP_PROFILE")
	authManager, err := auth.NewManager(profilePath)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup":
			yes := hasArg(os.Args[2:], "--yes") || hasArg(os.Args[2:], "-y")
			setupCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
			if err := runSetup(setupCtx, authManager, yes); err != nil {
				log.Fatal(err)
			}
			return
		case "install-browsers":
			if err := playwright.Install(); err != nil {
				log.Fatal(err)
			}
			fmt.Fprintln(os.Stderr, "Playwright browsers installed")
			return
		case "login":
			if len(os.Args) > 3 && os.Args[2] == "--cookie" {
				if err := auth.ImportCookieHeader(authManager.Store, strings.Join(os.Args[3:], " ")); err != nil {
					log.Fatal(err)
				}
				if _, err := authManager.RefreshTokens(ctx); err != nil {
					log.Fatalf("cookie saved but verification failed: %v", err)
				}
				fmt.Fprintln(os.Stderr, "NotebookLM cookie login saved")
				return
			}
			loginCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			if err := authManager.Login(loginCtx); err != nil {
				log.Fatal(err)
			}
			fmt.Fprintln(os.Stderr, "NotebookLM login saved")
			return
		case "version":
			fmt.Println(version)
			return
		case "--help", "-h", "help":
			fmt.Fprint(os.Stderr, usage)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
			os.Exit(2)
		}
	}

	rpcClient := rpc.NewClient(authManager)
	notebookLM := notebooklm.New(rpcClient)
	server := &mcp.Server{Name: "notebooklm-mcp-go", Version: version, Tools: mcp.NotebookLMTools(notebookLM)}
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
