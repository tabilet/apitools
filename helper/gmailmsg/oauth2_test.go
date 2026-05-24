package gmailmsg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleOAuth2CredentialFromMapAcceptsN8NTokenData(t *testing.T) {
	cred, err := GoogleOAuth2CredentialFromMap(map[string]any{
		"clientId":           "client-id",
		"clientSecret":       "client-secret",
		"oauthRedirectUrl":   "http://localhost/callback",
		"oauthTokenData":     map[string]any{"refresh_token": "refresh-token"},
		"authorization_url":  "https://accounts.google.com/o/oauth2/v2/auth",
		"token_url":          "https://oauth2.googleapis.com/token",
		"scopes":             []any{"https://www.googleapis.com/auth/gmail.send"},
		"ignored_plain_name": "ignored",
	})
	if err != nil {
		t.Fatalf("GoogleOAuth2CredentialFromMap error: %v", err)
	}
	if cred.ClientID != "client-id" || cred.ClientSecret != "client-secret" || cred.RefreshToken != "refresh-token" {
		t.Fatalf("credential fields = %#v", cred)
	}
	if len(cred.Scopes) != 1 || cred.Scopes[0] != "https://www.googleapis.com/auth/gmail.send" {
		t.Fatalf("scopes = %#v", cred.Scopes)
	}
}

func TestGoogleOAuth2CredentialRefreshesAccessToken(t *testing.T) {
	var request map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		request = map[string]string{}
		for key := range r.Form {
			request[key] = r.Form.Get(key)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	cred := GoogleOAuth2Credential{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RefreshToken: "refresh-token",
		TokenURL:     server.URL,
		Scopes:       []string{"scope-a"},
	}
	token, err := cred.AccessTokenValue(context.Background())
	if err != nil {
		t.Fatalf("AccessTokenValue error: %v", err)
	}
	if token != "access-token" {
		t.Fatalf("token = %q", token)
	}
	if request["grant_type"] != "refresh_token" || request["refresh_token"] != "refresh-token" {
		t.Fatalf("request = %#v", request)
	}
}

func TestGoogleOAuth2CredentialRequiresAuthorizationMaterial(t *testing.T) {
	cred := GoogleOAuth2Credential{
		ClientID:         "client-id",
		ClientSecret:     "client-secret",
		OAuthRedirectURL: "http://localhost/callback",
		Scopes:           []string{"scope-a"},
	}
	_, err := cred.AccessTokenValue(context.Background())
	if err == nil || !IsAuthorizationRequired(err) {
		t.Fatalf("expected authorization required error, got %v", err)
	}
	if !strings.Contains(err.Error(), "accounts.google.com") {
		t.Fatalf("error should include auth URL, got %v", err)
	}
}
