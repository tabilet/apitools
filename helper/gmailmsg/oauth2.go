package gmailmsg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	DefaultGoogleOAuth2AuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	DefaultGoogleOAuth2TokenURL = "https://oauth2.googleapis.com/token"
)

// GoogleOAuth2Credential is a runtime-supplied credential bootstrap shape for
// Gmail OAuth2. It is intentionally separate from fnct helpers: trusted
// runtimes may use it to mint a bearer token, but gmail.render_raw remains pure.
type GoogleOAuth2Credential struct {
	ClientID          string
	ClientSecret      string
	OAuthRedirectURL  string
	AuthorizationCode string
	RefreshToken      string
	AccessToken       string
	TokenExpiry       time.Time
	AuthURL           string
	TokenURL          string
	Scopes            []string
}

type AuthorizationRequiredError struct {
	AuthURL string
}

func (e *AuthorizationRequiredError) Error() string {
	if e == nil {
		return ""
	}
	if e.AuthURL == "" {
		return "google oauth2 authorization is required"
	}
	return "google oauth2 authorization is required; open the authorization URL and provide authorization_code or refresh_token in the runtime data file: " + e.AuthURL
}

func IsAuthorizationRequired(err error) bool {
	var target *AuthorizationRequiredError
	return errors.As(err, &target)
}

// GoogleOAuth2CredentialFromMap decodes a prompt-safe data-file map into a
// credential bootstrap shape. It accepts snake_case, camelCase, and n8n-style
// nested oauthTokenData fields.
func GoogleOAuth2CredentialFromMap(value any) (GoogleOAuth2Credential, error) {
	m, err := mapFromAny(value)
	if err != nil {
		return GoogleOAuth2Credential{}, err
	}
	tokenData, _ := mapValue(m, "oauth_token_data", "oauthTokenData", "token_data", "tokenData")
	out := GoogleOAuth2Credential{
		ClientID:          stringValue(m, "client_id", "clientId", "oauth_client_id", "oauthClientId"),
		ClientSecret:      stringValue(m, "client_secret", "clientSecret", "oauth_client_secret", "oauthClientSecret"),
		OAuthRedirectURL:  stringValue(m, "oauth_redirect_url", "oauthRedirectUrl", "oauthRedirectURL", "redirect_url", "redirectUrl", "redirect_uri", "redirectUri"),
		AuthorizationCode: stringValue(m, "authorization_code", "authorizationCode", "auth_code", "authCode", "code"),
		RefreshToken:      firstNonEmptyString(stringValue(m, "refresh_token", "refreshToken"), stringValue(tokenData, "refresh_token", "refreshToken")),
		AccessToken:       firstNonEmptyString(stringValue(m, "access_token", "accessToken"), stringValue(tokenData, "access_token", "accessToken")),
		AuthURL:           firstNonEmptyString(stringValue(m, "auth_url", "authUrl", "authorization_url", "authorizationUrl"), DefaultGoogleOAuth2AuthURL),
		TokenURL:          firstNonEmptyString(stringValue(m, "token_url", "tokenUrl", "token_uri", "tokenUri"), DefaultGoogleOAuth2TokenURL),
		Scopes:            stringSliceValue(m, "scopes", "oauth_scopes", "oauthScopes"),
	}
	if expiry := firstNonEmptyString(stringValue(m, "token_expiry", "tokenExpiry", "expiry"), stringValue(tokenData, "expiry", "expires_at", "expiresAt")); expiry != "" {
		if parsed, err := time.Parse(time.RFC3339, expiry); err == nil {
			out.TokenExpiry = parsed
		}
	}
	return out, nil
}

func (c GoogleOAuth2Credential) WithScopes(scopes []string) GoogleOAuth2Credential {
	if len(c.Scopes) == 0 {
		c.Scopes = append([]string(nil), scopes...)
		return c
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(c.Scopes)+len(scopes))
	for _, scope := range append(append([]string(nil), c.Scopes...), scopes...) {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, scope)
	}
	c.Scopes = out
	return c
}

func (c GoogleOAuth2Credential) AuthCodeURL(state string) (string, error) {
	cfg, err := c.oauth2Config(true)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(state) == "" {
		state = "openudon"
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce), nil
}

func (c GoogleOAuth2Credential) AccessTokenValue(ctx context.Context) (string, error) {
	if token := strings.TrimSpace(c.AccessToken); token != "" && (c.TokenExpiry.IsZero() || time.Until(c.TokenExpiry) > time.Minute) {
		return token, nil
	}
	cfg, err := c.oauth2Config(strings.TrimSpace(c.AuthorizationCode) != "")
	if err != nil {
		return "", err
	}
	switch {
	case strings.TrimSpace(c.RefreshToken) != "":
		token, err := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: strings.TrimSpace(c.RefreshToken)}).Token()
		if err != nil {
			return "", fmt.Errorf("refresh google oauth2 access token: %w", err)
		}
		if strings.TrimSpace(token.AccessToken) == "" {
			return "", errors.New("refresh google oauth2 access token returned an empty access token")
		}
		return token.AccessToken, nil
	case strings.TrimSpace(c.AuthorizationCode) != "":
		token, err := cfg.Exchange(ctx, strings.TrimSpace(c.AuthorizationCode))
		if err != nil {
			return "", fmt.Errorf("exchange google oauth2 authorization code: %w", err)
		}
		if strings.TrimSpace(token.AccessToken) == "" {
			return "", errors.New("exchange google oauth2 authorization code returned an empty access token")
		}
		return token.AccessToken, nil
	default:
		authURL, err := c.AuthCodeURL("openudon")
		if err != nil {
			return "", err
		}
		return "", &AuthorizationRequiredError{AuthURL: authURL}
	}
}

func (c GoogleOAuth2Credential) ExchangeAuthorizationCode(ctx context.Context) (*oauth2.Token, error) {
	if strings.TrimSpace(c.AuthorizationCode) == "" {
		return nil, errors.New("google oauth2 authorization_code is required")
	}
	cfg, err := c.oauth2Config(true)
	if err != nil {
		return nil, err
	}
	token, err := cfg.Exchange(ctx, strings.TrimSpace(c.AuthorizationCode))
	if err != nil {
		return nil, fmt.Errorf("exchange google oauth2 authorization code: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return nil, errors.New("exchange google oauth2 authorization code returned an empty access token")
	}
	return token, nil
}

func (c GoogleOAuth2Credential) oauth2Config(requireRedirect bool) (*oauth2.Config, error) {
	if strings.TrimSpace(c.ClientID) == "" {
		return nil, errors.New("google oauth2 client_id is required")
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		return nil, errors.New("google oauth2 client_secret is required")
	}
	if requireRedirect && strings.TrimSpace(c.OAuthRedirectURL) == "" {
		return nil, errors.New("google oauth2 oauth_redirect_url is required for authorization-code flow")
	}
	authURL := firstNonEmptyString(c.AuthURL, DefaultGoogleOAuth2AuthURL)
	tokenURL := firstNonEmptyString(c.TokenURL, DefaultGoogleOAuth2TokenURL)
	if _, err := url.ParseRequestURI(authURL); err != nil {
		return nil, fmt.Errorf("invalid google oauth2 auth_url: %w", err)
	}
	if _, err := url.ParseRequestURI(tokenURL); err != nil {
		return nil, fmt.Errorf("invalid google oauth2 token_url: %w", err)
	}
	return &oauth2.Config{
		ClientID:     strings.TrimSpace(c.ClientID),
		ClientSecret: strings.TrimSpace(c.ClientSecret),
		RedirectURL:  strings.TrimSpace(c.OAuthRedirectURL),
		Scopes:       append([]string(nil), c.Scopes...),
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}, nil
}

func mapFromAny(value any) (map[string]any, error) {
	if value == nil {
		return nil, errors.New("google oauth2 credential data is nil")
	}
	if m, ok := value.(map[string]any); ok {
		return m, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal google oauth2 credential data: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode google oauth2 credential data: %w", err)
	}
	return out, nil
}

func mapValue(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if typed, ok := value.(map[string]any); ok {
				return typed, true
			}
			if decoded, err := mapFromAny(value); err == nil {
				return decoded, true
			}
		}
	}
	return nil, false
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func stringSliceValue(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []string:
			return append([]string(nil), typed...)
		case []any:
			var out []string
			for _, item := range typed {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
			return out
		case string:
			var out []string
			for _, part := range strings.Split(typed, " ") {
				if part = strings.TrimSpace(part); part != "" {
					out = append(out, part)
				}
			}
			return out
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
