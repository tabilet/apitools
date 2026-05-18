package catalog

const (
	tryN8nSpecRoot = "../try-n8n/reducibility/specs/"
	n8nNodeRoot    = "../n8n/packages/nodes-base/nodes/"
)

var builtInCandidates = []Candidate{
	workflowCandidate(candidateSeed{
		id:          "asana",
		displayName: "Asana",
		aliases:     []string{"asana api"},
		category:    "project-management",
		n8nNode:     "Asana",
		relevance:   "Popular workflow service for task, project, and work-management automation.",
	}),
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
		id:          "box",
		displayName: "Box",
		aliases:     []string{"box api"},
		category:    "files",
		n8nNode:     "Box",
		relevance:   "Popular workflow service for file storage, sharing, and content-management automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "calendly",
		displayName: "Calendly",
		aliases:     []string{"calendly api"},
		category:    "scheduling",
		n8nNode:     "Calendly",
		relevance:   "Popular workflow service for scheduling events, invitees, and booking automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "clickup",
		displayName: "ClickUp",
		aliases:     []string{"clickup api", "click up"},
		category:    "project-management",
		n8nNode:     "ClickUp",
		relevance:   "Popular workflow service for tasks, lists, spaces, and team work-management automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "discord",
		displayName: "Discord",
		aliases:     []string{"discord api"},
		category:    "messaging",
		n8nNode:     "Discord",
		relevance:   "Popular workflow service for community messaging, channels, and notification automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "dropbox",
		displayName: "Dropbox",
		aliases:     []string{"dropbox api"},
		category:    "files",
		n8nNode:     "Dropbox",
		relevance:   "Popular workflow service for file storage, transfer, and sharing automation.",
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
		id:              "google-calendar",
		displayName:     "Google Calendar",
		aliases:         []string{"google calendar api", "calendar api"},
		category:        "scheduling",
		n8nNode:         "Google/Calendar",
		relevance:       "Popular workflow service for calendar events, availability, and scheduling automation.",
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
		id:              "google-sheets",
		displayName:     "Google Sheets",
		aliases:         []string{"sheets", "google sheets api"},
		category:        "spreadsheet",
		n8nNode:         "Google/Sheet",
		relevance:       "Popular workflow service for spreadsheet reads, writes, and tabular-data automation.",
		machineSpecKind: "google-discovery",
		machineStatus:   SpecStatusNeedsVerification,
		userNeed:        UserOpenAPINeedLikely,
	}),
	workflowCandidate(candidateSeed{
		id:          "github",
		displayName: "GitHub",
		aliases:     []string{"github api"},
		category:    "developer-tools",
		n8nNode:     "Github",
		relevance:   "Popular workflow service for repositories, issues, pull requests, and developer automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "gitlab",
		displayName: "GitLab",
		aliases:     []string{"gitlab api"},
		category:    "developer-tools",
		n8nNode:     "Gitlab",
		relevance:   "Popular workflow service for repositories, issues, merge requests, and DevOps automation.",
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
		id:          "microsoft-graph",
		displayName: "Microsoft Graph",
		aliases:     []string{"microsoft graph api", "graph api", "office 365"},
		category:    "productivity",
		n8nNode:     "Microsoft",
		relevance:   "Popular platform API for Microsoft 365 users, files, mail, calendars, and collaboration automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "notion",
		displayName: "Notion",
		aliases:     []string{"notion api"},
		category:    "knowledge-management",
		n8nNode:     "Notion",
		relevance:   "Popular workflow service for pages, databases, and knowledge-base automation.",
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
		id:          "salesforce",
		displayName: "Salesforce",
		aliases:     []string{"salesforce api", "salesforce crm"},
		category:    "crm",
		n8nNode:     "Salesforce",
		relevance:   "Popular CRM platform for leads, accounts, opportunities, and customer automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "shopify",
		displayName: "Shopify",
		aliases:     []string{"shopify api"},
		category:    "commerce",
		n8nNode:     "Shopify",
		relevance:   "Popular commerce platform for orders, products, customers, and storefront automation.",
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
		id:          "stripe",
		displayName: "Stripe",
		aliases:     []string{"stripe api"},
		category:    "payments",
		n8nNode:     "Stripe",
		relevance:   "Popular payment platform for customers, charges, invoices, subscriptions, and billing automation.",
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
	workflowCandidate(candidateSeed{
		id:          "twilio",
		displayName: "Twilio",
		aliases:     []string{"twilio api", "twilio sms"},
		category:    "communications",
		n8nNode:     "Twilio",
		relevance:   "Popular workflow service for SMS, voice, messaging, phone numbers, and communication automation.",
	}),
	workflowCandidate(candidateSeed{
		id:          "zendesk",
		displayName: "Zendesk",
		aliases:     []string{"zendesk api", "zendesk support"},
		category:    "customer-support",
		n8nNode:     "Zendesk",
		relevance:   "Popular workflow service for support tickets, users, organizations, messaging, and customer-service automation.",
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
	candidate := Candidate{
		ID:                        seed.id,
		DisplayName:               seed.displayName,
		Aliases:                   append([]string(nil), seed.aliases...),
		Category:                  seed.category,
		WorkflowRelevance:         seed.relevance,
		OfficialOpenAPIStatus:     SpecStatusNeedsVerification,
		OfficialMachineSpecStatus: machineStatus,
		OfficialMachineSpecKind:   seed.machineSpecKind,
		UserOpenAPINeed:           userNeed,
		AuthSecurityReview:        AuthSecurityNotReviewed,
	}
	if seed.fixture != "" {
		fixturePath := tryN8nSpecRoot + seed.fixture
		candidate.LocalOpenAPIFixture = fixturePath
		candidate.Evidence = append(candidate.Evidence, CandidateEvidence{
			Source: EvidenceTryN8nLocalFixture,
			Use:    EvidenceUseSeedFixture,
			Ref:    fixturePath,
			Note:   "Local OpenUdon advisory fixture; not treated as an official provider source.",
		})
	}
	if seed.n8nNode != "" {
		candidate.Evidence = append(candidate.Evidence, CandidateEvidence{
			Source: EvidenceN8nNodeDirectory,
			Use:    EvidenceUsePriority,
			Ref:    n8nNodeRoot + seed.n8nNode,
			Note:   "n8n built-in node presence is a prioritization signal only, not runtime compatibility evidence.",
		})
	}
	return candidate
}
