package catalog

import "testing"

func TestBuildN8nNodeGapReportClassifiesNodeRoots(t *testing.T) {
	report := BuildN8nNodeGapReport(N8nNodeGapReportOptions{
		NodeRoots: []string{
			"Slack",
			"Google",
			"MongoDb",
			"GraphQL",
			"Code",
			"MissingProvider",
			"nodes",
		},
		Providers: []Provider{
			{
				ID:                              "google-drive",
				DisplayName:                     "Google Drive",
				ReviewState:                     ProviderReviewedCatalogEntry,
				CandidateID:                     "google-drive",
				OfficialOpenAPIAvailability:     SpecAvailabilityUnknown,
				OfficialMachineSpecAvailability: SpecAvailabilityKnown,
				UserOpenAPINeed:                 UserOpenAPINeedLikely,
				SpecReferences: []SpecReference{
					{
						ID:              "google-drive-discovery",
						Kind:            SpecKindGoogleDiscovery,
						URL:             "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest",
						SourceAuthority: SourceAuthorityOfficialProvider,
						SourceNote:      "Official Discovery document.",
					},
				},
			},
			{
				ID:                              "slack",
				DisplayName:                     "Slack",
				Aliases:                         []string{"slack api"},
				ReviewState:                     ProviderReviewedCatalogEntry,
				CandidateID:                     "slack",
				OfficialOpenAPIAvailability:     SpecAvailabilityKnown,
				OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
				UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
				SpecReferences: []SpecReference{
					{
						ID:              "slack-openapi",
						Kind:            SpecKindOpenAPI,
						URL:             "https://api.slack.com/specs/openapi/v2/slack_web_openapi_v2_without_examples.json",
						SourceAuthority: SourceAuthorityOfficialProvider,
						SourceNote:      "Official OpenAPI document.",
					},
				},
			},
		},
	})
	if report.TotalNodes != 7 {
		t.Fatalf("TotalNodes = %d, want 7", report.TotalNodes)
	}
	tests := map[string]N8nNodeGapClassification{
		"Slack":           N8nNodeAlreadyCovered,
		"Google":          N8nNodeAlreadyCovered,
		"MongoDb":         N8nNodeGenericProtocolExcluded,
		"GraphQL":         N8nNodeProtocolFamilyCandidate,
		"Code":            N8nNodeLocalWorkflowUtility,
		"MissingProvider": N8nNodeProviderAPICandidate,
		"nodes":           N8nNodeInternalDirectoryExcluded,
	}
	for node, want := range tests {
		row, ok := findN8nGapRow(report.Rows, node)
		if !ok {
			t.Fatalf("missing row for %s", node)
		}
		if row.Classification != want {
			t.Fatalf("%s classification = %q, want %q", node, row.Classification, want)
		}
	}
	row, _ := findN8nGapRow(report.Rows, "Google")
	if row.MatchedProviderID != "google-drive" || row.MatchedBy != "category-coverage" {
		t.Fatalf("Google match = provider %q by %q, want google-drive category-coverage", row.MatchedProviderID, row.MatchedBy)
	}
	if len(report.FrozenBatch) != 10 {
		t.Fatalf("len(FrozenBatch) = %d, want 10", len(report.FrozenBatch))
	}
}

func TestBuildN8nNodeGapReportMatchesCompactAliases(t *testing.T) {
	report := BuildN8nNodeGapReport(N8nNodeGapReportOptions{
		NodeRoots: []string{"ActiveCampaign", "ApiTemplateIo", "Rocketchat"},
		Providers: []Provider{
			{
				ID:                              "activecampaign",
				DisplayName:                     "ActiveCampaign",
				ReviewState:                     ProviderReviewedCatalogEntry,
				CandidateID:                     "activecampaign",
				OfficialOpenAPIAvailability:     SpecAvailabilityUnknown,
				OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
				UserOpenAPINeed:                 UserOpenAPINeedLikely,
			},
			{
				ID:                              "apitemplate-io",
				DisplayName:                     "APITemplate.io",
				ReviewState:                     ProviderReviewedCatalogEntry,
				CandidateID:                     "apitemplate-io",
				OfficialOpenAPIAvailability:     SpecAvailabilityUnknown,
				OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
				UserOpenAPINeed:                 UserOpenAPINeedLikely,
			},
			{
				ID:                              "rocket-chat",
				DisplayName:                     "Rocket.Chat",
				ReviewState:                     ProviderReviewedCatalogEntry,
				CandidateID:                     "rocket-chat",
				OfficialOpenAPIAvailability:     SpecAvailabilityUnknown,
				OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
				UserOpenAPINeed:                 UserOpenAPINeedLikely,
			},
		},
	})
	for _, node := range []string{"ActiveCampaign", "ApiTemplateIo", "Rocketchat"} {
		row, ok := findN8nGapRow(report.Rows, node)
		if !ok {
			t.Fatalf("missing row for %s", node)
		}
		if row.Classification != N8nNodeAlreadyCovered {
			t.Fatalf("%s classification = %q, want already covered", node, row.Classification)
		}
	}
}

func findN8nGapRow(rows []N8nNodeGapRow, node string) (N8nNodeGapRow, bool) {
	for _, row := range rows {
		if row.NodeRoot == node {
			return row, true
		}
	}
	return N8nNodeGapRow{}, false
}
