package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenUdon/apitools/helper/gmailmsg"
)

func runOAuth(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		oauthUsage(out)
		return 2
	}
	switch args[0] {
	case "google":
		return runOAuthGoogle(args[1:], out, errOut)
	case "-h", "--help", "help":
		oauthUsage(out)
		return 0
	default:
		fmt.Fprintf(errOut, "unknown oauth command %q\n", args[0])
		oauthUsage(errOut)
		return 2
	}
}

func oauthUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: apitools oauth <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  google  run a local Google OAuth2 authorization-code login")
}

func runOAuthGoogle(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		oauthGoogleUsage(out)
		return 2
	}
	switch args[0] {
	case "login":
		return runOAuthGoogleLogin(args[1:], out, errOut)
	case "-h", "--help", "help":
		oauthGoogleUsage(out)
		return 0
	default:
		fmt.Fprintf(errOut, "unknown oauth google command %q\n", args[0])
		oauthGoogleUsage(errOut)
		return 2
	}
}

func oauthGoogleUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: apitools oauth google <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  login  run a local Google OAuth2 authorization-code login")
}

type oauthGoogleLoginConfig struct {
	Binding             string
	ClientID            string
	ClientSecretEnv     string
	ClientSecret        string
	Scopes              []string
	Listen              string
	RedirectURL         string
	AuthURL             string
	TokenURL            string
	Code                string
	RefreshTokenEnv     string
	ClientIDEnv         string
	Timeout             time.Duration
	AllowMissingRefresh bool
}

func runOAuthGoogleLogin(args []string, out, errOut io.Writer) int {
	cfg := oauthGoogleLoginConfig{}
	fs := flag.NewFlagSet("apitools oauth google login", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&cfg.Binding, "binding", "googleOAuth2", "Credential binding name for generated data.hcl snippet")
	fs.StringVar(&cfg.ClientID, "client-id", "", "Google OAuth2 client ID")
	fs.StringVar(&cfg.ClientSecretEnv, "client-secret-env", "GOOGLE_CLIENT_SECRET", "Environment variable containing the Google OAuth2 client secret")
	fs.Var((*stringListFlag)(&cfg.Scopes), "scope", "OAuth2 scope; may be repeated")
	fs.StringVar(&cfg.Listen, "listen", "127.0.0.1:8765", "Local callback listen address")
	fs.StringVar(&cfg.RedirectURL, "redirect-url", "", "OAuth2 redirect URL; defaults to http://<listen>/oauth2/callback")
	fs.StringVar(&cfg.AuthURL, "auth-url", gmailmsg.DefaultGoogleOAuth2AuthURL, "OAuth2 authorization endpoint")
	fs.StringVar(&cfg.TokenURL, "token-url", gmailmsg.DefaultGoogleOAuth2TokenURL, "OAuth2 token endpoint")
	fs.StringVar(&cfg.Code, "code", "", "Authorization code to exchange instead of starting a local callback server")
	fs.StringVar(&cfg.RefreshTokenEnv, "refresh-token-env", "GOOGLE_REFRESH_TOKEN", "Environment variable name to use for the refresh token in output")
	fs.StringVar(&cfg.ClientIDEnv, "client-id-env", "", "Optional environment variable name to use for client_id in the generated HCL snippet")
	fs.DurationVar(&cfg.Timeout, "timeout", 5*time.Minute, "Maximum time to wait for browser callback")
	fs.BoolVar(&cfg.AllowMissingRefresh, "allow-missing-refresh-token", false, "Allow success when Google returns only an access token")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools oauth google login --client-id <id> --scope <scope> [options]")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if hasHelpFlag(args) {
		fs.SetOutput(out)
		fs.Usage()
		return 0
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(errOut, "unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}
	if err := cfg.resolve(); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	token, redirectURL, err := executeOAuthGoogleLogin(context.Background(), cfg, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	writeOAuthGoogleLoginOutput(out, cfg, redirectURL, token.RefreshToken, token.AccessToken)
	if strings.TrimSpace(token.RefreshToken) == "" && !cfg.AllowMissingRefresh {
		fmt.Fprintln(errOut, "Google did not return a refresh token. Revoke the app grant or rerun with a new consent prompt, then try again.")
		return 1
	}
	return 0
}

func (c *oauthGoogleLoginConfig) resolve() error {
	c.Binding = strings.TrimSpace(c.Binding)
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.ClientSecretEnv = strings.TrimSpace(c.ClientSecretEnv)
	c.Listen = strings.TrimSpace(c.Listen)
	c.RedirectURL = strings.TrimSpace(c.RedirectURL)
	c.AuthURL = strings.TrimSpace(c.AuthURL)
	c.TokenURL = strings.TrimSpace(c.TokenURL)
	c.Code = strings.TrimSpace(c.Code)
	c.RefreshTokenEnv = strings.TrimSpace(c.RefreshTokenEnv)
	c.ClientIDEnv = strings.TrimSpace(c.ClientIDEnv)
	if c.Binding == "" {
		return fmt.Errorf("--binding is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("--client-id is required")
	}
	if c.ClientSecretEnv == "" {
		return fmt.Errorf("--client-secret-env is required")
	}
	secret, ok := os.LookupEnv(c.ClientSecretEnv)
	if !ok || strings.TrimSpace(secret) == "" {
		return fmt.Errorf("environment variable %s is not set", c.ClientSecretEnv)
	}
	c.ClientSecret = strings.TrimSpace(secret)
	if len(c.Scopes) == 0 {
		return fmt.Errorf("at least one --scope is required")
	}
	c.Scopes = normalizeScopes(c.Scopes)
	if c.Listen == "" && c.Code == "" {
		return fmt.Errorf("--listen is required when --code is not provided")
	}
	if c.AuthURL == "" {
		c.AuthURL = gmailmsg.DefaultGoogleOAuth2AuthURL
	}
	if c.TokenURL == "" {
		c.TokenURL = gmailmsg.DefaultGoogleOAuth2TokenURL
	}
	if c.RefreshTokenEnv == "" {
		return fmt.Errorf("--refresh-token-env is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	return nil
}

func executeOAuthGoogleLogin(ctx context.Context, cfg oauthGoogleLoginConfig, errOut io.Writer) (*oauth2Token, string, error) {
	redirectURL := cfg.RedirectURL
	code := cfg.Code
	var shutdown func(context.Context) error
	if code == "" {
		var err error
		code, redirectURL, shutdown, err = waitForOAuthCode(ctx, cfg, errOut)
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = shutdown(context.Background()) }()
	}
	cred := gmailmsg.GoogleOAuth2Credential{
		ClientID:          cfg.ClientID,
		ClientSecret:      cfg.ClientSecret,
		OAuthRedirectURL:  redirectURL,
		AuthorizationCode: code,
		AuthURL:           cfg.AuthURL,
		TokenURL:          cfg.TokenURL,
		Scopes:            cfg.Scopes,
	}
	token, err := cred.ExchangeAuthorizationCode(ctx)
	if err != nil {
		return nil, "", err
	}
	return &oauth2Token{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken}, redirectURL, nil
}

type oauth2Token struct {
	AccessToken  string
	RefreshToken string
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func waitForOAuthCode(ctx context.Context, cfg oauthGoogleLoginConfig, errOut io.Writer) (string, string, func(context.Context) error, error) {
	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return "", "", nil, fmt.Errorf("listen for oauth callback: %w", err)
	}
	redirectURL := cfg.RedirectURL
	if redirectURL == "" {
		redirectURL = "http://" + listener.Addr().String() + "/oauth2/callback"
	}
	state, err := randomState()
	if err != nil {
		_ = listener.Close()
		return "", "", nil, err
	}
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("state"); got != state {
			http.Error(w, "OAuth state did not match", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth state did not match"):
			default:
			}
			return
		}
		if providerErr := r.URL.Query().Get("error"); providerErr != "" {
			http.Error(w, "OAuth provider returned an error", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth provider returned error %q", providerErr):
			default:
			}
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			http.Error(w, "OAuth callback did not include code", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth callback did not include code"):
			default:
			}
			return
		}
		fmt.Fprintf(w, "<!doctype html><title>OAuth complete</title><p>%s</p>", html.EscapeString("OAuth complete. You can return to the terminal."))
		select {
		case codeCh <- code:
		default:
		}
	})
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()
	cred := gmailmsg.GoogleOAuth2Credential{
		ClientID:         cfg.ClientID,
		ClientSecret:     cfg.ClientSecret,
		OAuthRedirectURL: redirectURL,
		AuthURL:          cfg.AuthURL,
		TokenURL:         cfg.TokenURL,
		Scopes:           cfg.Scopes,
	}
	authURL, err := cred.AuthCodeURL(state)
	if err != nil {
		_ = server.Shutdown(context.Background())
		return "", "", nil, err
	}
	fmt.Fprintf(errOut, "OAuth redirect URL: %s\n", redirectURL)
	fmt.Fprintf(errOut, "Open this URL in your browser:\n%s\n", authURL)
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	select {
	case code := <-codeCh:
		return code, redirectURL, server.Shutdown, nil
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return "", "", nil, err
	case <-timeoutCtx.Done():
		_ = server.Shutdown(context.Background())
		return "", "", nil, fmt.Errorf("oauth login timed out after %s", cfg.Timeout)
	}
}

func writeOAuthGoogleLoginOutput(out io.Writer, cfg oauthGoogleLoginConfig, redirectURL, refreshToken, accessToken string) {
	fmt.Fprintf(out, "export %s=%s\n", cfg.RefreshTokenEnv, shellQuote(refreshToken))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "credentials {")
	fmt.Fprintf(out, "  %s {\n", cfg.Binding)
	if cfg.ClientIDEnv != "" {
		fmt.Fprintf(out, "    client_id = \"ENVIRONMENT:%s\"\n", cfg.ClientIDEnv)
	} else {
		fmt.Fprintf(out, "    client_id = %s\n", strconv.Quote(cfg.ClientID))
	}
	fmt.Fprintf(out, "    client_secret = \"ENVIRONMENT:%s\"\n", cfg.ClientSecretEnv)
	if redirectURL != "" {
		fmt.Fprintf(out, "    oauth_redirect_url = %s\n", strconv.Quote(redirectURL))
	}
	fmt.Fprintf(out, "    refresh_token = \"ENVIRONMENT:%s\"\n", cfg.RefreshTokenEnv)
	fmt.Fprintln(out, "  }")
	fmt.Fprintln(out, "}")
	if refreshToken == "" && accessToken != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "# No refresh token was returned; access token omitted from the HCL snippet because it expires.")
	}
}

func randomState() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func normalizeScopes(scopes []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, scope := range scopes {
		for _, part := range strings.Fields(scope) {
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	sort.Strings(out)
	return out
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
