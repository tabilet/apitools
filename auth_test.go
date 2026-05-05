package apitools

import (
	"reflect"
	"testing"
)

func TestAuthRequirementsForOperationAWSSigV4Fields(t *testing.T) {
	got := AuthRequirementsForOperation("aws", OperationSummary{
		OperationID: "GET_DescribeInstanceStatus",
		Security: []SecuritySummary{{
			Name:          "hmac",
			Type:          "apiKey",
			In:            "header",
			ParameterName: "Authorization",
		}},
	})
	if len(got) != 1 {
		t.Fatalf("requirements = %#v", got)
	}
	requirement := got[0]
	if requirement.Kind != "aws_signature" || requirement.Scheme != "hmac" {
		t.Fatalf("requirement = %#v", requirement)
	}
	if !reflect.DeepEqual(requirement.CredentialFields, []string{"aws_access_key_id", "aws_secret_access_key"}) {
		t.Fatalf("credential fields = %#v", requirement.CredentialFields)
	}
	if !reflect.DeepEqual(requirement.OptionalCredentialFields, []string{"aws_session_token"}) {
		t.Fatalf("optional credential fields = %#v", requirement.OptionalCredentialFields)
	}
	if !reflect.DeepEqual(requirement.ConfigFields, []string{"region", "service"}) {
		t.Fatalf("config fields = %#v", requirement.ConfigFields)
	}
}

func TestAuthRequirementsForOperationDetectsExplicitAWSSigV4Metadata(t *testing.T) {
	got := AuthRequirementsForOperation("generic", OperationSummary{
		Security: []SecuritySummary{{
			Name:   "sigv4Auth",
			Type:   "http",
			Scheme: "awsSigv4",
		}},
	})
	if len(got) != 1 || got[0].Kind != "aws_signature" {
		t.Fatalf("requirements = %#v", got)
	}
}

func TestAuthRequirementsForOperationGenericSchemes(t *testing.T) {
	got := AuthRequirementsForOperation("sandbox", OperationSummary{
		Security: []SecuritySummary{
			{Name: "apiKeyAuth", Type: "apiKey", In: "header", ParameterName: "X-API-Key"},
			{Name: "bearerAuth", Type: "http", Scheme: "bearer"},
			{Name: "basicAuth", Type: "http", Scheme: "basic"},
			{Name: "googleOAuth", Type: "oauth2", Flows: []string{"authorizationCode"}, Scopes: []string{"drive.readonly"}},
		},
	})
	byKind := map[string]AuthRequirementSummary{}
	for _, requirement := range got {
		byKind[requirement.Kind] = requirement
	}
	if !reflect.DeepEqual(byKind["api_key"].CredentialFields, []string{"x_api_key"}) {
		t.Fatalf("api key requirement = %#v", byKind["api_key"])
	}
	if !reflect.DeepEqual(byKind["bearer"].CredentialFields, []string{"access_token"}) {
		t.Fatalf("bearer requirement = %#v", byKind["bearer"])
	}
	if !reflect.DeepEqual(byKind["basic"].CredentialFields, []string{"username", "password"}) {
		t.Fatalf("basic requirement = %#v", byKind["basic"])
	}
	if byKind["oauth2"].Scheme != "googleOAuth" || len(byKind["oauth2"].CredentialFields) != 0 || !reflect.DeepEqual(byKind["oauth2"].Flows, []string{"authorizationCode"}) {
		t.Fatalf("oauth2 requirement = %#v", byKind["oauth2"])
	}
}

func TestAuthRequirementsForOperationDetectsGCPOAuth2Dialect(t *testing.T) {
	got := AuthRequirementsForOperation("gmail", OperationSummary{
		Security: []SecuritySummary{{
			Name:             "Oauth2c",
			Type:             "oauth2",
			Flows:            []string{"authorizationCode"},
			OAuthFlows:       []OAuthFlowSummary{{Name: "authorizationCode", AuthorizationURL: "https://accounts.google.com/o/oauth2/auth", TokenURL: "https://oauth2.googleapis.com/token", Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"}}},
			AuthorizationURL: "https://accounts.google.com/o/oauth2/auth",
			TokenURL:         "https://oauth2.googleapis.com/token",
			Scopes:           []string{"https://www.googleapis.com/auth/gmail.readonly"},
		}},
	})
	if len(got) != 1 {
		t.Fatalf("requirements = %#v", got)
	}
	requirement := got[0]
	if requirement.Kind != "oauth2" || requirement.Dialect != "gcp" {
		t.Fatalf("requirement = %#v", requirement)
	}
	if requirement.AuthorizationURL != "https://accounts.google.com/o/oauth2/auth" || requirement.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("oauth URLs = %#v", requirement)
	}
	if len(requirement.OAuthFlows) != 1 || requirement.OAuthFlows[0].Name != "authorizationCode" || requirement.OAuthFlows[0].TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("oauth flows = %#v", requirement.OAuthFlows)
	}
	if !reflect.DeepEqual(requirement.CredentialFields, []string{"oauth_client_id", "oauth_client_secret"}) {
		t.Fatalf("credential fields = %#v", requirement.CredentialFields)
	}
	if !reflect.DeepEqual(requirement.ConfigFields, []string{"oauth_auth_uri", "oauth_token_uri", "oauth_redirect_uri"}) {
		t.Fatalf("config fields = %#v", requirement.ConfigFields)
	}
}

func TestAuthRequirementsForOperationsMergesScopesDeterministically(t *testing.T) {
	got := AuthRequirementsForOperations("google", []OperationSummary{
		{OperationID: "z", Security: []SecuritySummary{{Name: "googleOAuth", Type: "oauth2", Flows: []string{"clientCredentials"}, Scopes: []string{"write", "read"}}}},
		{OperationID: "a", Security: []SecuritySummary{{Name: "googleOAuth", Type: "oauth2", Flows: []string{"authorizationCode"}, Scopes: []string{"read"}}}},
	})
	if len(got) != 1 {
		t.Fatalf("requirements = %#v", got)
	}
	if !reflect.DeepEqual(got[0].Flows, []string{"authorizationCode", "clientCredentials"}) {
		t.Fatalf("flows = %#v", got[0].Flows)
	}
	if !reflect.DeepEqual(got[0].Scopes, []string{"read", "write"}) {
		t.Fatalf("scopes = %#v", got[0].Scopes)
	}
}
