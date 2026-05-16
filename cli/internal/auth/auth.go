package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// YouTubeReadOnlyScope is the OAuth2 scope required to read YouTube data.
const YouTubeReadOnlyScope = "https://www.googleapis.com/auth/youtube.readonly"

// TokenManager handles storing and retrieving OAuth2 tokens.
type TokenManager struct {
	tokenPath string
	oauthConf *oauth2.Config
}

// New creates a TokenManager. cacheDir is typically ~/.headliner.
func New(clientID, clientSecret, cacheDir string) *TokenManager {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{YouTubeReadOnlyScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8085/callback",
	}
	return &TokenManager{
		tokenPath: filepath.Join(cacheDir, "token.json"),
		oauthConf: conf,
	}
}

// GetClient returns an authenticated HTTP client, running the browser OAuth
// consent flow if no valid token is cached.
func (tm *TokenManager) GetClient(ctx context.Context) (*http.Client, error) {
	tok, err := tm.loadToken()
	if err != nil || !tok.Valid() {
		tok, err = tm.runConsentFlow(ctx)
		if err != nil {
			return nil, err
		}
		if err := tm.saveToken(tok); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not cache token: %v\n", err)
		}
	}
	return tm.oauthConf.Client(ctx, tok), nil
}

// runConsentFlow starts a local HTTP server, opens the browser for OAuth
// consent, and waits for the authorization code redirect.
func (tm *TokenManager) runConsentFlow(ctx context.Context) (*oauth2.Token, error) {
	state := fmt.Sprintf("headliner-%d", time.Now().UnixNano())
	authURL := tm.oauthConf.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{Addr: ":8085"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("invalid state parameter")
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "no code", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "<html><body><h2>✅ Authorization successful — you can close this tab.</h2></body></html>")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Printf("\n🔐 Opening browser for YouTube authorization...\n")
	fmt.Printf("   If the browser doesn't open, visit:\n   %s\n\n", authURL)
	openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, fmt.Errorf("OAuth2 consent flow failed: %w", err)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for authorization")
	}

	// Shutdown local server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	tok, err := tm.oauthConf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}
	return tok, nil
}

func (tm *TokenManager) loadToken() (*oauth2.Token, error) {
	f, err := os.Open(tm.tokenPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

func (tm *TokenManager) saveToken(tok *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(tm.tokenPath), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(tm.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

// openBrowser tries to launch the system's default browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return
	}
	_ = exec.Command(cmd, args...).Start()
}
