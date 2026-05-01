package apitools

import "testing"

func TestContainsLikelyCredentialValueFlagsCommonTokenFamilies(t *testing.T) {
	values := []string{
		"AIzaabcdefghijklmnopqrstuvwxyz1234567890",
		"ghp_abcdefghijklmnopqrstuvwxyzABCDEFGHIJ",
		"sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890",
		"sk-proj-abcdefghijklmnopqrstuvwxyz1234567890",
		"AKIA1234567890ABCDEF",
		"Bearer abcdefghijklmnop1234567890",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ",
		`api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"`,
		`appid = "abc123abc123abc123"`,
		`app_id = "abc123abc123abc123"`,
		`password = "abc123abc123abc123"`,
	}
	for _, value := range values {
		if !ContainsLikelyCredentialValue([]byte(value)) {
			t.Fatalf("credential scanner did not flag %q", value)
		}
	}
}

func TestContainsLikelyCredentialValueAllowsWorkflowReferencesAndBindings(t *testing.T) {
	values := []string{
		`from = "inputs.ticketId"`,
		`to = "get_ticket.received_body.requesterEmail"`,
		`subject = "get_ticket.received_body.subject"`,
		`body = "get_ticket.received_body.summary"`,
		`lat = "get_coordinates.received_body[0].lat"`,
		`appid = "weather_appid"`,
		`api_key = "weather_api_key"`,
		`token_from = "weather_api_key"`,
		`authorization = "inputs.authorization"`,
		`get_ticket.received_body.requesterEmail`,
		`weather_api_key`,
		`weather_appid`,
	}
	for _, value := range values {
		if ContainsLikelyCredentialValue([]byte(value)) {
			t.Fatalf("credential scanner flagged valid reference or binding %q", value)
		}
	}
}

func TestScanCredentialValuesReportsArtifactPaths(t *testing.T) {
	diagnostics := ScanCredentialValues([]Artifact{
		{Path: "symbolic.hcl", Content: []byte(`api_key = "runtime.support_api"`)},
		{Path: "secret.hcl", Content: []byte(`api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"`)},
	})
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if diagnostics[0].Path != "secret.hcl" || diagnostics[0].Code != "leaf.literal_credential" {
		t.Fatalf("diagnostic = %#v", diagnostics[0])
	}
}
