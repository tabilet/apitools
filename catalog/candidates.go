package catalog

const (
	tryN8nSpecRoot = "../try-n8n/reducibility/specs/"
	n8nNodeRoot    = "../n8n/packages/nodes-base/nodes/"
)

var builtInCandidates = []Candidate{
	workflowCandidate(candidateSeed{
		id:          "airtable",
		displayName: "Airtable",
		aliases:     []string{"airtable api"},
		category:    "database",
		fixture:     "airtable.json",
		n8nNode:     "Airtable",
		relevance:   "Common workflow source and destination for records in bases and tables.",
	}),
	workflowCandidate(candidateSeed{
		id:              "gmail",
		displayName:     "Gmail",
		aliases:         []string{"google mail", "gmail api"},
		category:        "email",
		fixture:         "gmail.json",
		n8nNode:         "Google/Gmail",
		relevance:       "Common workflow service for reading, sending, and routing email messages.",
		machineSpecKind: "google-discovery",
		machineStatus:   SpecStatusNeedsVerification,
		userNeed:        UserOpenAPINeedLikely,
	}),
	workflowCandidate(candidateSeed{
		id:              "google-drive",
		displayName:     "Google Drive",
		aliases:         []string{"drive", "google drive api"},
		category:        "files",
		fixture:         "google_drive.json",
		n8nNode:         "Google/Drive",
		relevance:       "Common workflow service for file storage, upload, search, and sharing.",
		machineSpecKind: "google-discovery",
		machineStatus:   SpecStatusNeedsVerification,
		userNeed:        UserOpenAPINeedLikely,
	}),
	workflowCandidate(candidateSeed{
		id:          "hubspot",
		displayName: "HubSpot",
		aliases:     []string{"hubspot crm", "hubspot api"},
		category:    "crm",
		fixture:     "hubspot.json",
		n8nNode:     "Hubspot",
		relevance:   "Common workflow service for CRM records, tickets, contacts, and pipeline automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "jira-cloud",
		displayName: "Jira Cloud",
		aliases:     []string{"jira", "atlassian jira", "jira api"},
		category:    "project-management",
		fixture:     "jira.json",
		n8nNode:     "Jira",
		relevance:   "Common workflow service for issue tracking and project automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "openweathermap",
		displayName: "OpenWeatherMap",
		aliases:     []string{"open weather map", "openweathermap api"},
		category:    "weather",
		fixture:     "openweathermap.json",
		n8nNode:     "OpenWeatherMap",
		relevance:   "Common workflow data source for current weather and forecast enrichment.",
	}),
	workflowCandidate(candidateSeed{
		id:          "pagerduty",
		displayName: "PagerDuty",
		aliases:     []string{"pagerduty api"},
		category:    "incident-management",
		fixture:     "pagerduty.json",
		n8nNode:     "PagerDuty",
		relevance:   "Common workflow service for incident, escalation, and on-call automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "slack",
		displayName: "Slack",
		aliases:     []string{"slack api"},
		category:    "messaging",
		fixture:     "slack.json",
		n8nNode:     "Slack",
		relevance:   "Common workflow service for messages, channels, and collaboration notifications.",
	}),
	workflowCandidate(candidateSeed{
		id:          "trello",
		displayName: "Trello",
		aliases:     []string{"trello api"},
		category:    "project-management",
		fixture:     "trello.json",
		n8nNode:     "Trello",
		relevance:   "Common workflow service for boards, lists, cards, and task automation.",
	}),
}

type candidateSeed struct {
	id              string
	displayName     string
	aliases         []string
	category        string
	fixture         string
	n8nNode         string
	relevance       string
	machineSpecKind string
	machineStatus   SpecStatus
	userNeed        UserOpenAPINeed
}

func workflowCandidate(seed candidateSeed) Candidate {
	machineStatus := seed.machineStatus
	if machineStatus == "" {
		machineStatus = SpecStatusUnknown
	}
	userNeed := seed.userNeed
	if userNeed == "" {
		userNeed = UserOpenAPINeedPossible
	}
	fixturePath := tryN8nSpecRoot + seed.fixture
	return Candidate{
		ID:                        seed.id,
		DisplayName:               seed.displayName,
		Aliases:                   append([]string(nil), seed.aliases...),
		Category:                  seed.category,
		WorkflowRelevance:         seed.relevance,
		LocalOpenAPIFixture:       fixturePath,
		OfficialOpenAPIStatus:     SpecStatusNeedsVerification,
		OfficialMachineSpecStatus: machineStatus,
		OfficialMachineSpecKind:   seed.machineSpecKind,
		UserOpenAPINeed:           userNeed,
		AuthSecurityReview:        AuthSecurityNotReviewed,
		Evidence: []CandidateEvidence{
			{
				Source: EvidenceTryN8nLocalFixture,
				Use:    EvidenceUseSeedFixture,
				Ref:    fixturePath,
				Note:   "Local OpenUdon advisory fixture; not treated as an official provider source.",
			},
			{
				Source: EvidenceN8nNodeDirectory,
				Use:    EvidenceUsePriority,
				Ref:    n8nNodeRoot + seed.n8nNode,
				Note:   "n8n built-in node presence is a prioritization signal only, not runtime compatibility evidence.",
			},
		},
	}
}
