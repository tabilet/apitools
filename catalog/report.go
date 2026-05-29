package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityClassification records source-backed auth/security completeness for
// providers whose upstream machine specs are already sufficient or still need
// review.
type SecurityClassification struct {
	ProviderID string                 `json:"provider_id"`
	SpecRefID  string                 `json:"spec_ref_id,omitempty"`
	Status     AuthCompletenessStatus `json:"status"`
	SourceRefs []string               `json:"source_refs,omitempty"`
	SourceNote string                 `json:"source_note"`
}

// SecurityReport is a deterministic metadata-only auth/security report for a
// provider catalog.
type SecurityReport struct {
	Providers []ProviderSecurityReport `json:"providers,omitempty"`
}

// ProviderSecurityReport records the effective catalog security status for one
// provider.
type ProviderSecurityReport struct {
	ProviderID  string                 `json:"provider_id"`
	Status      AuthCompletenessStatus `json:"status"`
	OverlayIDs  []string               `json:"overlay_ids,omitempty"`
	SpecRefIDs  []string               `json:"spec_ref_ids,omitempty"`
	SourceRefs  []string               `json:"source_refs,omitempty"`
	SourceNotes []string               `json:"source_notes,omitempty"`
}

// BuiltInSecurityClassifications returns source-backed security status records
// for built-in providers. Callers receive independent copies.
func BuiltInSecurityClassifications() []SecurityClassification {
	out := cloneSecurityClassifications(builtInSecurityClassifications)
	sortSecurityClassifications(out)
	return out
}

// BuiltInSecurityReport returns the built-in provider security report.
func BuiltInSecurityReport() (SecurityReport, error) {
	return BuildSecurityReport(BuiltInProviders(), BuiltInSecurityOverlays(), BuiltInSecurityClassifications())
}

// BuildSecurityReport combines provider metadata, security overlays, and
// source-backed classifications into a deterministic report.
func BuildSecurityReport(providers []Provider, overlays []SecurityOverlay, classifications []SecurityClassification) (SecurityReport, error) {
	if err := ValidateProviders(providers); err != nil {
		return SecurityReport{}, err
	}
	if err := ValidateSecurityOverlays(overlays, providers); err != nil {
		return SecurityReport{}, err
	}
	overlays = cloneSecurityOverlays(overlays)
	sortSecurityOverlays(overlays)
	providerByID := map[string]Provider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}

	classificationByProvider := map[string]SecurityClassification{}
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providerByID); err != nil {
			return SecurityReport{}, fmt.Errorf("security classification[%d]: %w", i, err)
		}
		if _, exists := classificationByProvider[classification.ProviderID]; exists {
			return SecurityReport{}, fmt.Errorf("security classification %q: duplicate provider", classification.ProviderID)
		}
		classificationByProvider[classification.ProviderID] = classification
	}

	overlaysByProvider := map[string][]SecurityOverlay{}
	for _, overlay := range overlays {
		overlaysByProvider[overlay.ProviderID] = append(overlaysByProvider[overlay.ProviderID], overlay)
	}

	var reports []ProviderSecurityReport
	for _, provider := range cloneProviders(providers) {
		report := ProviderSecurityReport{
			ProviderID: provider.ID,
			Status:     AuthStatusUnknown,
		}
		if classification, ok := classificationByProvider[provider.ID]; ok {
			report.Status = classification.Status
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, classification.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, classification.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, classification.SourceNote)
		}
		for _, overlay := range overlaysByProvider[provider.ID] {
			report.Status = overlay.Status
			report.OverlayIDs = appendIfNotEmpty(report.OverlayIDs, overlay.ID)
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, overlay.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, overlay.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, overlay.SourceNote)
		}
		report.OverlayIDs = sortedUniqueStrings(report.OverlayIDs)
		report.SpecRefIDs = sortedUniqueStrings(report.SpecRefIDs)
		report.SourceRefs = sortedUniqueStrings(report.SourceRefs)
		report.SourceNotes = sortedUniqueStrings(report.SourceNotes)
		reports = append(reports, report)
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].ProviderID < reports[j].ProviderID
	})
	return SecurityReport{Providers: reports}, nil
}

// FindProvider returns a provider security report by provider ID.
func (r SecurityReport) FindProvider(providerID string) (ProviderSecurityReport, bool) {
	normalized := normalizeKey(providerID)
	for _, provider := range r.Providers {
		if normalizeKey(provider.ProviderID) == normalized {
			return provider, true
		}
	}
	return ProviderSecurityReport{}, false
}

var builtInSecurityClassifications = []SecurityClassification{
	{
		ProviderID: "action-network",
		SpecRefID:  "action-network-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://actionnetwork.org/docs/v2/"},
		SourceNote: "Action Network has official REST API v2 human docs but no recorded stable public official OpenAPI document; OSDI-API-Token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "amplitude",
		SpecRefID:  "amplitude-api-authentication",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://amplitude.com/docs/apis/authentication", "https://amplitude.com/docs/apis/keys-and-tokens"},
		SourceNote: "Amplitude has official API docs but no recorded stable provider-wide official OpenAPI artifact; family-specific Basic, api_key, and bearer-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "anthropic",
		SpecRefID:  "anthropic-api-authentication",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://platform.claude.com/docs/en/api/authentication", "https://platform.claude.com/docs/en/api/messages"},
		SourceNote: "Anthropic has official API docs but no recorded stable public OpenAPI artifact; x-api-key metadata comes from advisory overlay notes and no prompt-execution endpoint overlay is registered.",
	},
	{
		ProviderID: "ashby",
		SpecRefID:  "ashby-api-authentication",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.ashbyhq.com/docs/authentication", "https://developers.ashbyhq.com/docs/introduction"},
		SourceNote: "Ashby has official human API docs but no recorded stable provider-wide OpenAPI artifact; Basic API-key auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "attio",
		SpecRefID:  "attio-rest-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.attio.com/openapi/api", "https://docs.attio.com/rest-api/endpoint-reference/openapi"},
		SourceNote: "Attio's official OpenAPI document includes oauth2 security scheme metadata and a root security requirement.",
	},
	{
		ProviderID: "acumatica",
		SpecRefID:  "acumatica-rest-api-openapi-docs",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://help.acumatica.com/Wiki/ShowWiki.aspx?pageid=6ff6b2c9-554d-4e65-b916-da2237a421f6", "https://help.acumatica.com/Wiki/ShowWiki.aspx?PageID=91dda8ed-5e92-48a5-a176-9a255506d0d6&wikiname=HelpRoot_Dev_Integration"},
		SourceNote: "Acumatica REST Swagger/OpenAPI metadata is generated by ERP instances and endpoints; authentication, tenants, companies, and endpoint versions are instance-specific and not complete in provider-wide metadata.",
	},
	{
		ProviderID: "adalo",
		SpecRefID:  "adalo-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.adalo.com/integrations/the-adalo-api/collections", "https://help.adalo.com/integrations/the-adalo-api/push-notifications"},
		SourceNote: "Adalo has official API human docs and app-specific generated collection docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "adobe-acrobat-sign",
		SpecRefID:  "adobe-acrobat-sign-openapi-sdk-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.adobe.com/acrobat-sign/docs/overview/sdks/openapi", "https://github.com/adobe/acrobat-sign/tree/main/sdks/AcrobatSign_OpenAPI_SDK"},
		SourceNote: "Adobe Acrobat Sign official SDK JSON files are Swagger 1.2-style rather than directly importable OpenAPI 2/3; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "adyen",
		SpecRefID:  "adyen-checkout-service-v72-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/Adyen/adyen-openapi/main/json/CheckoutService-v72.json", "https://docs.adyen.com/development-resources/api-credentials/"},
		SourceNote: "Adyen publishes official OpenAPI documents and API credential docs, but credential placement, merchant accounts, API families, and payment operation permissions need downstream review before binding credentials.",
	},
	{
		ProviderID: "affinity",
		SpecRefID:  "affinity-v1-api-reference",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api-docs.affinity.co/"},
		SourceNote: "Affinity has official V1 API human docs but no recorded stable public official OpenAPI document; basic and bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "aftership",
		SpecRefID:  "aftership-tracking-api-overview",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.aftership.com/docs/tracking", "https://www.aftership.com/docs/tracking/fcd9acb5f448a-api-overview", "https://www.aftership.com/docs/tracking/2024-07/quickstart/authentication"},
		SourceNote: "AfterShip Tracking docs advertise an OAS export, but M57 direct artifact probes were not usable as a durable unauthenticated catalog fetch; as-api-key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "agile-crm",
		SpecRefID:  "agile-crm-rest-api-github",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.agilecrm.com/api", "https://github.com/agilecrm/rest-api"},
		SourceNote: "Agile CRM has official API human docs and REST API GitHub documentation but no recorded stable public official OpenAPI document; email/API-key basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "aircall",
		SpecRefID:  "aircall-public-api-reference",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.aircall.io/api-references/"},
		SourceNote: "Aircall has official Public API human docs but no recorded stable public official OpenAPI document; Basic Auth and OAuth2 public_api metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "airwallex",
		SpecRefID:  "airwallex-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.airwallex.com/docs/developer-tools/api", "https://www.airwallex.com/docs/api/authentication/api_access/login", "https://www.airwallex.com/docs/developer-tools/api/manage-api-keys"},
		SourceNote: "Airwallex has official API human docs but no recorded stable provider-wide OpenAPI document; access-token bearer metadata comes from advisory overlay notes, with endpoint overlay deferred for financial movement safety.",
	},
	{
		ProviderID: "braze",
		SpecRefID:  "braze-api-basics",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.braze.com/docs/api/basics", "https://www.braze.com/docs/api/endpoints/user_data/post_user_track"},
		SourceNote: "Braze has official REST API docs but no recorded stable provider-wide official OpenAPI artifact; bearer REST API key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "canva",
		SpecRefID:  "canva-connect-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://www.canva.dev/sources/connect/api/latest/api.yml", "https://www.canva.dev/docs/connect/authentication/"},
		SourceNote: "Canva's official Connect API OpenAPI document includes oauthAuthCode security scheme metadata and operation security requirements.",
	},
	{
		ProviderID: "fivetran",
		SpecRefID:  "fivetran-rest-api-v1-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://fivetran.com/assets-docs/openapi/file_v1.json", "https://fivetran.com/docs/rest-api/getting-started"},
		SourceNote: "Fivetran's official REST API OpenAPI document includes basicAuth security scheme metadata and root security requirements.",
	},
	{
		ProviderID: "klaviyo",
		SpecRefID:  "klaviyo-stable-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/klaviyo/openapi/main/openapi/stable.json", "https://developers.klaviyo.com/en/reference"},
		SourceNote: "Klaviyo's official stable OpenAPI document includes Klaviyo-API-Key security scheme metadata and root security requirements.",
	},
	{
		ProviderID: "launchdarkly",
		SpecRefID:  "launchdarkly-rest-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://app.launchdarkly.com/api/v2/openapi.json", "https://launchdarkly.com/docs/api"},
		SourceNote: "LaunchDarkly's official REST API OpenAPI document includes ApiKey security scheme metadata and root security requirements.",
	},
	{
		ProviderID: "postman",
		SpecRefID:  "postman-api-authentication-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://learning.postman.com/docs/developer/postman-api/authentication/", "https://learning.postman.com/docs/developer/postman-api/intro-api/"},
		SourceNote: "Postman has official API docs but no recorded stable provider-owned OpenAPI artifact; X-API-Key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "apollo",
		SpecRefID:  "apollo-authentication-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.apollo.io/reference/authentication.md", "https://docs.apollo.io/reference/people-api-search.md", "https://docs.apollo.io/reference/organization-search.md"},
		SourceNote: "Apollo docs expose endpoint-level OpenAPI fragments and API-key/OAuth metadata but no recorded stable provider-wide OpenAPI document; x-api-key and bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "asana",
		SpecRefID:  "asana-openapi-v1",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml"},
		SourceNote: "Asana's official OpenAPI document includes bearer personal access token and OAuth2 security schemes plus root and operation security requirements.",
	},
	{
		ProviderID: "auth0",
		SpecRefID:  "auth0-management-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://auth0.com/docs/api/management/openapi.json", "https://auth0.com/docs/secure/tokens/access-tokens/management-api-access-tokens"},
		SourceNote: "Auth0 Management API publishes an official OpenAPI schema with bearer and OAuth2 client-credentials security metadata. Tenant domains, scopes, API clients, and access-token issuance remain downstream.",
	},
	{
		ProviderID: "airtable",
		SpecRefID:  "airtable-web-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://airtable.com/developers/web/api/introduction"},
		SourceNote: "Airtable has no recorded official OpenAPI document in the built-in catalog; security metadata must come from advisory overlay notes when importing a user-provided spec.",
	},
	{
		ProviderID: "checkr",
		SpecRefID:  "checkr-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.checkr.com/"},
		SourceNote: "Checkr has official Redoc-rendered human API docs but no recorded stable public OpenAPI JSON URL; basic_auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "bannerbear",
		SpecRefID:  "bannerbear-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.bannerbear.com/v2/"},
		SourceNote: "Bannerbear has official API v2 docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "baserow",
		SpecRefID:  "baserow-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.baserow.io/api/schema.json", "https://baserow.io/docs/apis/rest-api", "https://baserow.io/user-docs/database-api"},
		SourceNote: "Baserow's official OpenAPI document includes database-token, JWT, and user-source JWT bearer security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "beeminder",
		SpecRefID:  "beeminder-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.beeminder.com/", "https://www.beeminder.com/api/v1/auth_token.json", "https://www.beeminder.com/apps/new"},
		SourceNote: "Beeminder has official API human docs but no recorded stable public official OpenAPI document; personal auth_token and OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "bigcommerce",
		SpecRefID:  "bigcommerce-catalog-products-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/bigcommerce/api-specs/main/reference/catalog/products_catalog.v3.yml", "https://docs.bigcommerce.com/docs/start/authentication"},
		SourceNote: "BigCommerce publishes official OpenAPI specs with X-Auth-Token API-key security metadata. Store hashes, OAuth/private-app credentials, scopes, and channel availability remain downstream.",
	},
	{
		ProviderID: "aws-s3",
		SpecRefID:  "aws-s3-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/s3/service/2006-03-01/s3-2006-03-01.json", "https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html"},
		SourceNote: "AWS S3 has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-lambda",
		SpecRefID:  "aws-lambda-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/lambda/service/2015-03-31/lambda-2015-03-31.json", "https://docs.aws.amazon.com/lambda/latest/api/welcome.html"},
		SourceNote: "AWS Lambda has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-sns",
		SpecRefID:  "aws-sns-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/sns/service/2010-03-31/sns-2010-03-31.json", "https://docs.aws.amazon.com/sns/latest/api/welcome.html"},
		SourceNote: "AWS SNS has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-api-gateway",
		SpecRefID:  "aws-api-gateway-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/api-gateway/service/2015-07-09/api-gateway-2015-07-09.json", "https://docs.aws.amazon.com/apigateway/latest/api/Welcome.html"},
		SourceNote: "AWS API Gateway has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-athena",
		SpecRefID:  "aws-athena-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/athena/service/2017-05-18/athena-2017-05-18.json", "https://docs.aws.amazon.com/athena/latest/APIReference/Welcome.html"},
		SourceNote: "AWS Athena has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-bedrock",
		SpecRefID:  "aws-bedrock-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/bedrock/service/2023-04-20/bedrock-2023-04-20.json", "https://docs.aws.amazon.com/bedrock/latest/APIReference/welcome.html"},
		SourceNote: "AWS Bedrock has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-cloudwatch",
		SpecRefID:  "aws-cloudwatch-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/cloudwatch/service/2010-08-01/cloudwatch-2010-08-01.json", "https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/Welcome.html"},
		SourceNote: "AWS CloudWatch has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-ec2",
		SpecRefID:  "aws-ec2-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/ec2/service/2016-11-15/ec2-2016-11-15.json", "https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Welcome.html"},
		SourceNote: "AWS EC2 has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-glue",
		SpecRefID:  "aws-glue-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/glue/service/2017-03-31/glue-2017-03-31.json", "https://docs.aws.amazon.com/glue/latest/webapi/Welcome.html"},
		SourceNote: "AWS Glue has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-guardduty",
		SpecRefID:  "aws-guardduty-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/guardduty/service/2017-11-28/guardduty-2017-11-28.json", "https://docs.aws.amazon.com/guardduty/latest/APIReference/Welcome.html"},
		SourceNote: "AWS GuardDuty has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-inspector2",
		SpecRefID:  "aws-inspector2-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/inspector2/service/2020-06-08/inspector2-2020-06-08.json", "https://docs.aws.amazon.com/inspector/v2/APIReference/Welcome.html"},
		SourceNote: "AWS Inspector2 has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-kinesis",
		SpecRefID:  "aws-kinesis-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/kinesis/service/2013-12-02/kinesis-2013-12-02.json", "https://docs.aws.amazon.com/kinesis/latest/APIReference/Welcome.html"},
		SourceNote: "AWS Kinesis has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-kms",
		SpecRefID:  "aws-kms-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/kms/service/2014-11-01/kms-2014-11-01.json", "https://docs.aws.amazon.com/kms/latest/APIReference/Welcome.html"},
		SourceNote: "AWS KMS has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-rds",
		SpecRefID:  "aws-rds-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/rds/service/2014-10-31/rds-2014-10-31.json", "https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/Welcome.html"},
		SourceNote: "AWS RDS has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-sagemaker",
		SpecRefID:  "aws-sagemaker-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/sagemaker/service/2017-07-24/sagemaker-2017-07-24.json", "https://docs.aws.amazon.com/sagemaker/latest/APIReference/Welcome.html"},
		SourceNote: "AWS SageMaker has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-secrets-manager",
		SpecRefID:  "aws-secrets-manager-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/secrets-manager/service/2017-10-17/secrets-manager-2017-10-17.json", "https://docs.aws.amazon.com/secretsmanager/latest/apireference/Welcome.html"},
		SourceNote: "AWS Secrets Manager has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-securityhub",
		SpecRefID:  "aws-securityhub-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/securityhub/service/2018-10-26/securityhub-2018-10-26.json", "https://docs.aws.amazon.com/securityhub/1.0/APIReference/Welcome.html"},
		SourceNote: "AWS Security Hub has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "kubernetes",
		SpecRefID:  "kubernetes-v1-19-2-swagger",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/hashicorp/terraform-provider-kubernetes/dcdf46c9ca238b671d1159f252ec19c8fe2ed16e/manifest/openapi/testdata/k8s-swagger.json", "https://kubernetes.io/docs/concepts/overview/kubernetes-api/", "https://kubernetes.io/docs/reference/access-authn-authz/authentication/", "https://kubernetes.io/docs/concepts/security/controlling-access/"},
		SourceNote: "The pinned Kubernetes v1.19.2 Swagger snapshot declares bearer-token authentication, but Kubernetes API authentication and authorization are cluster-configured; real cluster exports have no portable provider-wide security scheme, and apitools must not read kubeconfig, resolve credentials, or contact API servers.",
	},
	{
		ProviderID: "box",
		SpecRefID:  "box-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/box/box-openapi/main/openapi.json"},
		SourceNote: "Box's official OpenAPI document includes an OAuth2 authorization code security scheme and root security requirement.",
	},
	{
		ProviderID: "bitbucket",
		SpecRefID:  "bitbucket-cloud-swagger-v2",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.bitbucket.org/swagger.json", "https://developer.atlassian.com/cloud/bitbucket/rest/intro/"},
		SourceNote: "Bitbucket Cloud's official Swagger/OpenAPI schema includes OAuth2, basic, and API key security definitions with operation security metadata.",
	},
	{
		ProviderID: "brandfetch",
		SpecRefID:  "brandfetch-brand-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.brandfetch.com/openapi.json", "https://docs.brandfetch.com/brand-api/overview"},
		SourceNote: "Brandfetch's official Brand API OpenAPI document includes bearer HTTP security metadata with operation-level security requirements.",
	},
	{
		ProviderID: "calendly",
		SpecRefID:  "calendly-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.calendly.com/api-docs"},
		SourceNote: "Calendly's official API reference is human documentation backed by Stoplight hosting in this catalog entry; no downloadable official OpenAPI document is recorded, so endpoint and security metadata come from advisory overlays.",
	},
	{
		ProviderID: "circleci",
		SpecRefID:  "circleci-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://circleci.com/api/v2/openapi.json", "https://circleci.com/docs/api/v2/", "https://circleci.com/docs/guides/toolkit/managing-api-tokens/"},
		SourceNote: "CircleCI's official API v2 OpenAPI document includes Circle-Token header, basic-auth, and deprecated query-token security schemes with root security requirements.",
	},
	{
		ProviderID: "clearbit",
		SpecRefID:  "clearbit-prospector-zapier-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.clearbit.com/hc/en-us/articles/6480449602967-Integrate-Clearbit-Prospector-with-Google-Sheets-Using-Zapier", "https://help.clearbit.com/hc/en-us/articles/6045527495191-How-Do-I-Access-My-Clearbit-API-Key"},
		SourceNote: "Clearbit has official API support docs but no recorded stable public official OpenAPI document; secret API-key basic-auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cloudflare",
		SpecRefID:  "cloudflare-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml", "https://developers.cloudflare.com/api/"},
		SourceNote: "Cloudflare's official OpenAPI document includes API token, API key/email, and user service key security schemes with root and operation security requirements.",
	},
	{
		ProviderID: "databricks",
		SpecRefID:  "databricks-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.databricks.com/api/workspace/introduction", "https://docs.databricks.com/aws/en/dev-tools/auth"},
		SourceNote: "Databricks has no recorded public official OpenAPI document for the full REST API in this catalog entry; auth metadata comes from advisory bearer-token overlay notes.",
	},
	{
		ProviderID: "clickup",
		SpecRefID:  "clickup-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://developer.clickup.com/openapi/clickup-api-v2-reference.json"},
		SourceNote: "ClickUp's official OpenAPI document includes an Authorization header API key scheme and root security, but OAuth authorization-code flow details are documented separately rather than fully modeled in the security scheme.",
	},
	{
		ProviderID: "cisco-meraki",
		SpecRefID:  "cisco-meraki-dashboard-api-v1-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/meraki/openapi/master/openapi/spec3.json", "https://developer.cisco.com/meraki/api-v1/"},
		SourceNote: "Cisco Meraki publishes an official Dashboard API OpenAPI document with API-key and bearer security metadata. Organization/network permissions and API-key handling remain dashboard-account specific.",
	},
	{
		ProviderID: "confluence-cloud",
		SpecRefID:  "confluence-cloud-rest-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://developer.atlassian.com/cloud/confluence/swagger.v3.json", "https://developer.atlassian.com/cloud/confluence/oauth-2-3lo-apps/"},
		SourceNote: "Atlassian publishes an official Confluence Cloud OpenAPI document, but app scopes, site tenancy, user permissions, and OAuth consent remain downstream concerns.",
	},
	{
		ProviderID: "confluent-cloud",
		SpecRefID:  "confluent-cloud-org-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/confluentinc/ccloud-sdk-go-v2/master/org/v2/api/openapi.yaml", "https://docs.confluent.io/cloud/current/api.html/"},
		SourceNote: "Confluent Cloud publishes official OpenAPI documents with Basic API-key and OAuth2 security metadata. Environments, clusters, API keys, service accounts, and data-plane endpoint selection remain tenant-specific.",
	},
	{
		ProviderID: "discord",
		SpecRefID:  "discord-api-v10-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json"},
		SourceNote: "Discord's official OpenAPI v10 preview document includes BotToken and OAuth2 security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "docker-engine",
		SpecRefID:  "docker-engine-api-v1-54-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.docker.com/reference/api/engine/version/v1.54.yaml", "https://docs.docker.com/reference/api/engine/"},
		SourceNote: "Docker Engine's official Swagger document describes local daemon operations but does not declare a portable OpenAPI security scheme; socket/TLS/SSH access and X-Registry-Auth handling are deployment-specific metadata only.",
	},
	{
		ProviderID: "docker-hub",
		SpecRefID:  "docker-hub-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.docker.com/reference/api/hub/latest.yaml", "https://docs.docker.com/security/access-tokens/"},
		SourceNote: "Docker Hub's official OpenAPI document includes bearer JWT security metadata and documents access-token creation; apitools records token requirements without logging in or creating tokens.",
	},
	{
		ProviderID: "docker-registry",
		SpecRefID:  "docker-registry-hub-supported-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.docker.com/reference/api/registry/latest.yaml", "https://docs.docker.com/reference/api/registry/auth/"},
		SourceNote: "Docker's registry OpenAPI document models Authorization bearer headers as endpoint parameters for the Docker Hub-supported subset; the challenge/token exchange flow is documented separately and remains metadata only.",
	},
	{
		ProviderID: "docusign",
		SpecRefID:  "docusign-esignature-rest-v2-1-swagger",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/docusign/OpenAPI-Specifications/master/esignature.rest.swagger-v2.1.json", "https://developers.docusign.com/platform/auth/"},
		SourceNote: "DocuSign publishes official Swagger/OpenAPI specs, but OAuth consent, account selection, scopes, and environment hosts must be handled downstream before credential binding.",
	},
	{
		ProviderID: "discourse",
		SpecRefID:  "discourse-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.discourse.org/openapi.json", "https://docs.discourse.org/", "https://github.com/discourse/discourse_api_docs"},
		SourceNote: "Discourse's official OpenAPI document is importable but currently lacks OpenAPI securitySchemes; Api-Key and Api-Username header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "dropbox",
		SpecRefID:  "dropbox-api-stone-spec",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/dropbox/dropbox-api-spec"},
		SourceNote: "Dropbox's official machine-readable source is a Stone spec, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping for OpenAPI-only consumers.",
	},
	{
		ProviderID: "copper",
		SpecRefID:  "copper-developer-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.copper.com/introduction/authentication.html", "https://developer.copper.com/"},
		SourceNote: "Copper has official Developer API human docs but no recorded stable public official OpenAPI document; token and user-email header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "elastic",
		SpecRefID:  "elastic-elasticsearch-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://www.elastic.co/docs/api/doc/elasticsearch.json", "https://www.elastic.co/docs/api/doc/elasticsearch/authentication"},
		SourceNote: "Elastic's official Elasticsearch OpenAPI document includes API key, basic, and bearer security schemes with root security requirements.",
	},
	{
		ProviderID: "github",
		SpecRefID:  "github-rest-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"},
		SourceNote: "GitHub's official OpenAPI document omits security schemes and requirements; official REST authentication docs need advisory bearer/basic overlay mapping.",
	},
	{
		ProviderID: "grafana",
		SpecRefID:  "grafana-http-api-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/grafana/grafana/main/public/openapi3.json", "https://grafana.com/docs/grafana/latest/developer-resources/api-reference/"},
		SourceNote: "Grafana's official HTTP API OpenAPI v3 document includes API-key Authorization header and basic auth security schemes with root security requirements.",
	},
	{
		ProviderID: "grist",
		SpecRefID:  "grist-rest-api-usage-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://support.getgrist.com/rest-api/", "https://support.getgrist.com/api/"},
		SourceNote: "Grist has official REST API usage and reference docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "freshservice",
		SpecRefID:  "freshservice-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.freshservice.com/", "https://support.freshservice.com/support/solutions/articles/50000012704-working-with-apis-in-freshservice"},
		SourceNote: "Freshservice has official API v2 human docs but no recorded stable public official OpenAPI document; API-key Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "gitlab",
		SpecRefID:  "gitlab-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml"},
		SourceNote: "GitLab's official Swagger 2.0 document includes token security definitions but no root security; official REST auth docs describe additional bearer-token forms.",
	},
	{
		ProviderID: "gong",
		SpecRefID:  "gong-api-access-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.gong.io/docs/receive-access-to-the-api", "https://help.gong.io/docs/create-an-app-for-gong", "https://help.gong.io/hc/en-us/articles/360046818511-Uploading-calls-from-a-non-integrated-telephony-system"},
		SourceNote: "Gong has official API human docs but no recorded stable public official OpenAPI document; access key/secret Basic auth and OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "gmail",
		SpecRefID:  "gmail-discovery-v1",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://gmail.googleapis.com/$discovery/rest?version=v1"},
		SourceNote: "Gmail's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-calendar",
		SpecRefID:  "calendar-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest"},
		SourceNote: "Google Calendar's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-drive",
		SpecRefID:  "drive-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.googleapis.com/discovery/v1/apis/drive/v3/rest"},
		SourceNote: "Google Drive's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-sheets",
		SpecRefID:  "sheets-discovery-v4",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://sheets.googleapis.com/$discovery/rest?version=v4"},
		SourceNote: "Google Sheets' official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "hubspot",
		SpecRefID:  "hubspot-public-api-spec-index",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.hubapi.com/public/api/spec/v1/specs"},
		SourceNote: "HubSpot publishes split official OpenAPI specs through an index; selected specs should be reviewed with auth overlay metadata before downstream use.",
	},
	{
		ProviderID: "intercom",
		SpecRefID:  "intercom-api-v2-15-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.intercom.com/_bundle/docs/references/%402.15/rest-api/api.intercom.io.json?download=", "https://developers.intercom.com/docs/references/rest-api/api.intercom.io"},
		SourceNote: "Intercom's official API v2.15 OpenAPI document includes bearer-token security schemes and root security requirements.",
	},
	{
		ProviderID: "jira-cloud",
		SpecRefID:  "jira-cloud-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json"},
		SourceNote: "Atlassian's Jira Cloud OpenAPI document includes OAuth2/basic security schemes and operation-level security metadata.",
	},
	{
		ProviderID: "jenkins",
		SpecRefID:  "jenkins-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.jenkins.io/doc/book/using/remote-access-api/", "https://www.jenkins.io/blog/2018/07/02/new-api-token-system/"},
		SourceNote: "Jenkins has no recorded official OpenAPI document in this catalog entry; auth metadata comes from advisory basic/API-token and crumb overlay notes.",
	},
	{
		ProviderID: "linear",
		SpecRefID:  "linear-graphql-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://linear.app/developers/graphql", "https://linear.app/docs/api/"},
		SourceNote: "Linear's official public API is GraphQL and no OpenAPI document is recorded in this catalog entry; personal API key and OAuth bearer-token metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "mailchimp",
		SpecRefID:  "mailchimp-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://mailchimp.com/developer/marketing/api/", "https://mailchimp.com/developer/marketing/guides/quick-start/"},
		SourceNote: "Mailchimp's official Marketing API docs describe OpenAPI-backed endpoint documentation and API-key authentication, but no stable public downloadable OpenAPI document is recorded in this catalog entry; security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "marketo",
		SpecRefID:  "marketo-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/rest-api", "https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/lead-database/leads", "https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/lead-database/activities"},
		SourceNote: "Adobe Marketo Engage has official REST human docs but no recorded stable public official OpenAPI document; Authorization bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-graph",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml"},
		SourceNote: "Microsoft Graph's official OpenAPI v1.0 document lacks OpenAPI security schemes; official auth docs need advisory bearer-token overlay mapping.",
	},
	{
		ProviderID: "microsoft-entra",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/identity-network-access-overview/"},
		SourceNote: "Microsoft Entra is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-excel",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/api/resources/excel"},
		SourceNote: "Microsoft Excel workbook APIs are cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-graph-security",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/api/resources/security-api-overview"},
		SourceNote: "Microsoft Graph Security is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-onedrive",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/onedrive-concept-overview"},
		SourceNote: "Microsoft OneDrive is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-outlook",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/api/resources/mail-api-overview"},
		SourceNote: "Microsoft Outlook mail/calendar APIs are cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-sharepoint",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/api/resources/sharepoint"},
		SourceNote: "Microsoft SharePoint is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-teams",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/teams-concept-overview"},
		SourceNote: "Microsoft Teams is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-todo",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml", "https://learn.microsoft.com/en-us/graph/api/resources/todo-overview"},
		SourceNote: "Microsoft To Do is cataloged as a service-level Microsoft Graph source. The shared Graph OpenAPI artifact lacks OpenAPI security schemes, so Microsoft identity platform OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "azure-cosmos-db",
		SpecRefID:  "azure-cosmos-db-resource-manager-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/cosmos-db/resource-manager/Microsoft.DocumentDB/DocumentDB/stable/2025-10-15/cosmos-db.json", "https://learn.microsoft.com/en-us/rest/api/cosmos-db/access-control-on-cosmosdb-resources"},
		SourceNote: "Azure Cosmos DB REST authorization requires Azure/shared-key metadata that must remain advisory; apitools does not compute signatures, resolve keys, or choose accounts, databases, tenants, or subscriptions.",
	},
	{
		ProviderID: "azure-storage",
		SpecRefID:  "azure-blob-storage-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/storage/data-plane/Microsoft.BlobStorage/stable/2025-11-05/blob.json", "https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-azure-active-directory", "https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key"},
		SourceNote: "Azure Storage Blob supports Microsoft Entra bearer tokens and shared-key signatures; apitools records auth choices as metadata only and does not fetch tokens, sign requests, or choose storage accounts.",
	},
	{
		ProviderID: "monday-com",
		SpecRefID:  "monday-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.monday.com/api-reference/docs", "https://developer.monday.com/api-reference/docs/getting-started"},
		SourceNote: "Monday.com's official API is GraphQL and no OpenAPI document is recorded in this catalog entry; Authorization-header token metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "netlify",
		SpecRefID:  "netlify-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://open-api.netlify.com/openapi.json", "https://docs.netlify.com/api/get-started/"},
		SourceNote: "Netlify's official OpenAPI document includes OAuth2 security metadata and root security requirements; official API docs describe bearer personal access tokens for manual requests.",
	},
	{
		ProviderID: "netsuite",
		SpecRefID:  "netsuite-suitetalk-rest-overview",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/chapter_1540391670.html", "https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/chapter_1540810168.html", "https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/article_0627022005.html"},
		SourceNote: "NetSuite SuiteTalk REST authentication is account and integration-record configured; account-specific OpenAPI metadata must be user-provided, and apitools must not fetch tokens, sign TBA requests, or call metadata-catalog URLs.",
	},
	{
		ProviderID: "notion",
		SpecRefID:  "notion-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.notion.com/openapi.json"},
		SourceNote: "Notion's official OpenAPI document includes bearer auth security scheme metadata and a root security requirement.",
	},
	{
		ProviderID: "okta",
		SpecRefID:  "okta-management-minimal-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/okta/okta-management-openapi-spec/master/dist/current/management-minimal.yaml", "https://developer.okta.com/docs/api/openapi/okta-management/guides/overview/"},
		SourceNote: "Okta's official Management OpenAPI document includes SSWS API-token and OAuth2 security schemes with operation-level security metadata; official docs recommend scoped OAuth2 access tokens where possible.",
	},
	{
		ProviderID: "openweathermap",
		SpecRefID:  "openweathermap-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://openweathermap.org/api"},
		SourceNote: "OpenWeather has no recorded official OpenAPI document in the built-in catalog; API key placement is captured by advisory overlay metadata.",
	},
	{
		ProviderID: "oracle-fusion-cloud-applications",
		SpecRefID:  "oracle-fusion-common-rest-docs",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.oracle.com/en/cloud/saas/applications-common/25b/farca/toc.htm", "https://docs.oracle.com/en/cloud/saas/applications-common/25b/farca/Access_Metadata.html", "https://docs.oracle.com/en/cloud/saas/human-resources/farws/rest-endpoints.html"},
		SourceNote: "Oracle Fusion Cloud Applications REST auth and `/describe` metadata depend on tenant, product family, enabled modules, version, and security context; apitools records source guidance only and must not contact Fusion tenants or resolve credentials.",
	},
	{
		ProviderID: "pagerduty",
		SpecRefID:  "pagerduty-rest-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/PagerDuty/api-schema/main/reference/REST/openapiv3.json"},
		SourceNote: "PagerDuty's official OpenAPI document includes an Authorization header API key scheme and root security requirement.",
	},
	{
		ProviderID: "paypal",
		SpecRefID:  "paypal-checkout-orders-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/checkout_orders_v2.json"},
		SourceNote: "PayPal's official Checkout Orders v2 OpenAPI document includes OAuth2 security scheme metadata and operation-level security requirements.",
	},
	{
		ProviderID: "pipedrive",
		SpecRefID:  "pipedrive-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.pipedrive.com/docs/api/v1/openapi-v2.yaml"},
		SourceNote: "Pipedrive's official API v2 OpenAPI document includes API-token, OAuth2, and basic auth security schemes plus operation-level security requirements.",
	},
	{
		ProviderID: "quickbooks",
		SpecRefID:  "quickbooks-online-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api", "https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization/oauth-2.0"},
		SourceNote: "QuickBooks Online has official REST API and OAuth 2.0 docs but no stable public downloadable OpenAPI document recorded in this catalog entry; OAuth bearer security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "salesforce",
		SpecRefID:  "salesforce-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm"},
		SourceNote: "Salesforce has no recorded stable public downloadable OpenAPI document for the core REST API in the built-in catalog; OAuth bearer security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "seatable",
		SpecRefID:  "seatable-authentication-v6-2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.seatable.com/openapi", "https://api.seatable.com/reference/authentication", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb8e", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb93", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb96", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb95"},
		SourceNote: "SeaTable's official v6.2 OpenAPI documents include separate bearer security schemes and operation security requirements for account, API, and base tokens.",
	},
	{
		ProviderID: "sendgrid",
		SpecRefID:  "sendgrid-mail-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/twilio/sendgrid-oai/main/spec/json/tsg_mail_v3.json"},
		SourceNote: "Twilio SendGrid's official Mail v3 OpenAPI document includes bearer API-key security metadata and root security requirements.",
	},
	{
		ProviderID: "sentry",
		SpecRefID:  "sentry-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.sentry.io/api/", "https://docs.sentry.io/api/auth/"},
		SourceNote: "Sentry has no recorded official OpenAPI document in this catalog entry; auth metadata comes from advisory bearer-token overlay notes.",
	},
	{
		ProviderID: "servicenow",
		SpecRefID:  "servicenow-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/c_RESTAPI.html", "https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/export-openapi-specification.html"},
		SourceNote: "ServiceNow documents per-instance OpenAPI export through REST API Explorer, but this catalog entry has no stable public downloadable OpenAPI document; Basic and OAuth security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "shopify",
		SpecRefID:  "shopify-admin-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://shopify.dev/docs/api/admin-rest"},
		SourceNote: "Shopify has no recorded official OpenAPI document in the built-in catalog; Admin REST access-token security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "slack",
		SpecRefID:  "slack-web-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json"},
		SourceNote: "Slack's archived official OpenAPI document includes OAuth metadata and operation security but needs freshness review against current Slack docs.",
	},
	{
		ProviderID: "snowflake",
		SpecRefID:  "snowflake-sql-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/snowflakedb/snowflake-rest-api-specs/main/specifications/sqlapi.yaml", "https://docs.snowflake.com/en/developer-guide/snowflake-rest-api/snowflake-rest-api"},
		SourceNote: "Snowflake's official SQL API OpenAPI document includes key-pair JWT, external OAuth, Snowflake OAuth, and programmatic access token security schemes with root and operation security requirements.",
	},
	{
		ProviderID: "splunk",
		SpecRefID:  "splunk-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.splunk.com/Documentation/Splunk/latest/RESTREF", "https://docs.splunk.com/Documentation/Splunk/latest/RESTUM/RESTusing"},
		SourceNote: "Splunk has no recorded stable public OpenAPI document for the general Enterprise REST API in this catalog entry; auth metadata comes from advisory token/basic overlay notes.",
	},
	{
		ProviderID: "stripe",
		SpecRefID:  "stripe-latest-openapi-spec3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/stripe/openapi/master/latest/openapi.spec3.json"},
		SourceNote: "Stripe's official latest OpenAPI document includes basic and bearer HTTP security schemes plus root security requirements.",
	},
	{
		ProviderID: "supabase",
		SpecRefID:  "supabase-management-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://api.supabase.com/api/v1-json", "https://supabase.com/docs/reference/api/introduction"},
		SourceNote: "Supabase's official Management API OpenAPI document includes a bearer security scheme but no root security requirement; official docs state bearer authorization is required for API requests.",
	},
	{
		ProviderID: "telegram",
		SpecRefID:  "telegram-bot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://core.telegram.org/bots/api"},
		SourceNote: "Telegram has official Bot API docs but no OpenAPI document recorded in this catalog entry; bot-token path authentication needs advisory metadata and cannot be represented exactly as an OpenAPI security scheme.",
	},
	{
		ProviderID: "trello",
		SpecRefID:  "trello-cloud-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/trello/swagger.v3.json"},
		SourceNote: "Atlassian's Trello OpenAPI document includes query API key/token security schemes and root security requirements.",
	},
	{
		ProviderID: "twilio",
		SpecRefID:  "twilio-api-v2010-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_api_v2010.json"},
		SourceNote: "Twilio's official OpenAPI v2010 document includes an HTTP Basic security scheme and operation-level security requirements for Account SID/Auth Token authentication.",
	},
	{
		ProviderID: "typeform",
		SpecRefID:  "typeform-developer-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.typeform.com/developers/get-started/", "https://www.typeform.com/developers/get-started/personal-access-token/", "https://www.typeform.com/developers/get-started/applications/"},
		SourceNote: "Typeform has official REST API and token/OAuth docs but no stable public downloadable OpenAPI document recorded in this catalog entry; bearer-token security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "xero",
		SpecRefID:  "xero-accounting-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/XeroAPI/Xero-OpenAPI/master/xero_accounting.yaml"},
		SourceNote: "Xero's official Accounting OpenAPI document includes OAuth2 authorization code security metadata and operation-level scope requirements.",
	},
	{
		ProviderID: "zoom",
		SpecRefID:  "zoom-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zoom/api/master/openapi.v2.json", "https://developers.zoom.us/docs/integrations/oauth/"},
		SourceNote: "Zoom's official GitHub Swagger 2.0 document includes older access_token query security, while current docs describe OAuth bearer access tokens and granular scopes; security metadata should be reviewed with advisory overlay notes.",
	},
	{
		ProviderID: "zendesk",
		SpecRefID:  "zendesk-sunshine-conversations-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zendesk/sunshine-conversations-api-spec/master/openapi.yaml"},
		SourceNote: "Zendesk's official Sunshine Conversations OpenAPI document includes basic and bearer HTTP security schemes plus root and operation security requirements for the messaging API surface.",
	},
	{
		ProviderID: "acuity-scheduling",
		SpecRefID:  "acuity-quick-start",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.acuityscheduling.com/docs/quick-start", "https://developers.acuityscheduling.com/reference"},
		SourceNote: "Acuity Scheduling has official API docs but no recorded official OpenAPI document; Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "bitly",
		SpecRefID:  "bitly-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dev.bitly.com/v4/v4.json", "https://dev.bitly.com/api-reference"},
		SourceNote: "Bitly's official API v4 OpenAPI document includes a bearer HTTP security scheme and root security requirement.",
	},
	{
		ProviderID: "brevo",
		SpecRefID:  "brevo-api-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.brevo.com/v3/swagger_definition_v3.yml", "https://developers.brevo.com/docs/api-clients"},
		SourceNote: "Brevo's official API v3 OpenAPI document includes api-key and partner-key header security schemes with root security requirements.",
	},
	{
		ProviderID: "clockify",
		SpecRefID:  "clockify-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.clockify.me/", "https://clockify.me/help/getting-started/clockify-api-overview"},
		SourceNote: "Clockify has official REST-shaped API docs but no recorded official OpenAPI document; X-Api-Key and X-Addon-Token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "coda",
		SpecRefID:  "coda-api-v1-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://coda.io/apis/v1/openapi.json", "https://coda.io/developers/apis/v1"},
		SourceNote: "Coda's official API v1 OpenAPI document includes a bearer HTTP security scheme and root security requirement.",
	},
	{
		ProviderID: "deepl",
		SpecRefID:  "deepl-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/DeepLcom/openapi/main/openapi.yaml", "https://developers.deepl.com/docs/resources/open-api-spec"},
		SourceNote: "DeepL's official OpenAPI document includes an Authorization header API-key scheme and operation security requirements.",
	},
	{
		ProviderID: "figma",
		SpecRefID:  "figma-rest-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/figma/rest-api-spec/main/openapi/openapi.yaml", "https://developers.figma.com/docs/rest-api/"},
		SourceNote: "Figma's official REST API OpenAPI document includes personal access token, plan access token, and OAuth2 security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "ghost",
		SpecRefID:  "ghost-admin-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.ghost.org/admin-api/", "https://docs.ghost.org/content-api/"},
		SourceNote: "Ghost has official Admin and Content API docs but no recorded official OpenAPI document; Admin JWT and Content API key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "harvest",
		SpecRefID:  "harvest-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.getharvest.com/api-v2/", "https://help.getharvest.com/api-v2/authentication-api/authentication/authentication/"},
		SourceNote: "Harvest has official API v2 docs but no recorded official OpenAPI document; bearer token plus Harvest-Account-ID metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "help-scout",
		SpecRefID:  "help-scout-mailbox-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.helpscout.com/mailbox-api/", "https://developer.helpscout.com/docs-api/"},
		SourceNote: "Help Scout has official Inbox API and Docs API docs but no recorded official OpenAPI document; OAuth and Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "iterable",
		SpecRefID:  "iterable-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.iterable.com/api/docs", "https://support.iterable.com/hc/en-us/articles/360043464871-API-Keys"},
		SourceNote: "Iterable has official API docs but no recorded official OpenAPI document; Api-Key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "convertkit",
		SpecRefID:  "kit-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.kit.com/api-reference/v4.json", "https://developers.kit.com/api-reference/authentication"},
		SourceNote: "Kit's official API V4 OpenAPI document includes API-key and OAuth2 security schemes with operation security requirements; official auth docs describe the X-Kit-Api-Key header for API-key access.",
	},
	{
		ProviderID: "mailjet",
		SpecRefID:  "mailjet-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://documentation.mailjet.com/hc/en-us/articles/360044088173-REST-API", "https://documentation.mailjet.com/hc/en-us/articles/360043225693-What-is-an-API-key"},
		SourceNote: "Mailjet has official REST API docs but no recorded official OpenAPI document; API key and secret key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "nocodb",
		SpecRefID:  "nocodb-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://nocodb.com/apis/v2/swagger-v2.json", "https://nocodb.com/apis/v3/swagger-v3.json", "https://nocodb.com/docs/product-docs/developer-resources/rest-apis/accessing-apis"},
		SourceNote: "NocoDB's official OpenAPI documents include API-token and auth-token metadata, but token requirements are split between reusable header parameters and security schemes, with v2/v3 differences that need normalization before treating auth as complete.",
	},
	{
		ProviderID: "posthog",
		SpecRefID:  "posthog-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://us.posthog.com/api/schema/", "https://posthog.com/docs/api"},
		SourceNote: "PostHog's official OpenAPI schema includes personal API key bearer authentication for private API endpoints; official API docs also describe public project-token ingestion endpoints that are not fully modeled by that schema.",
	},
	{
		ProviderID: "uptimerobot",
		SpecRefID:  "uptimerobot-api-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://cdn.uptimerobot.com/api/openapi.yaml", "https://uptimerobot.com/api/v3/"},
		SourceNote: "UptimeRobot's official API v3 OpenAPI document includes a bearer security scheme with operation-level security requirements; official v3 docs describe API-key access and rate limits.",
	},
	{
		ProviderID: "coingecko",
		SpecRefID:  "coingecko-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.coingecko.com/reference/authentication", "https://docs.coingecko.com/reference/endpoint-overview"},
		SourceNote: "CoinGecko has official API human docs but no recorded stable public official OpenAPI document; Demo and Pro API-key header/query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "hackernews",
		SpecRefID:  "hackernews-firebase-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://github.com/HackerNews/API"},
		SourceNote: "Hacker News publishes an official public Firebase API for item, user, story-list, maxitem, and update reads without authentication.",
	},
	{
		ProviderID: "marketstack",
		SpecRefID:  "marketstack-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.swaggerhub.com/apis/apilayer-863/MarketstackAPIv2/2.0.0/swagger.json", "https://docs.apilayer.com/marketstack/docs/marketstack-api-v2-v-2-0-0"},
		SourceNote: "Marketstack's official API v2 OpenAPI document includes an access_key query API-key security scheme with a root security requirement.",
	},
	{
		ProviderID: "nasa",
		SpecRefID:  "nasa-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.nasa.gov/"},
		SourceNote: "NASA Open APIs have official human docs but no recorded stable public official OpenAPI document; shared api_key query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "onesimpleapi",
		SpecRefID:  "onesimpleapi-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://onesimpleapi.com/docs", "https://onesimpleapi.com/user/api-tokens"},
		SourceNote: "OneSimpleApi has official human docs but no recorded stable public official OpenAPI document; token query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "openthesaurus",
		SpecRefID:  "openthesaurus-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://www.openthesaurus.de/about/api"},
		SourceNote: "OpenThesaurus official API docs describe anonymous GET synonym lookup in JSON or XML, subject to published usage conditions.",
	},
	{
		ProviderID: "quickchart",
		SpecRefID:  "quickchart-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://quickchart.io/documentation/", "https://quickchart.io/documentation/qr-codes/"},
		SourceNote: "QuickChart official chart and QR-code API docs describe anonymous rendering endpoints in the reviewed scope; paid/account features are outside this catalog row.",
	},
	{
		ProviderID: "reddit",
		SpecRefID:  "reddit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.reddit.com/dev/api/", "https://www.reddit.com/dev/api/oauth", "https://developers.reddit.com/docs/capabilities/server/reddit-api"},
		SourceNote: "Reddit has official generated API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata for oauth.reddit.com comes from advisory overlay notes.",
	},
	{
		ProviderID: "autopilot",
		SpecRefID:  "autopilot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://autopilot.docs.apiary.io/", "https://help.ortto.com/a-376-autopilot-how-to-use-autopilots-api"},
		SourceNote: "Autopilot has official human API docs but no recorded stable public official OpenAPI document; legacy API-key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "drift",
		SpecRefID:  "drift-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://devdocs.drift.com/docs/authentication-and-scopes", "https://devdocs.drift.com/docs/platform-apis"},
		SourceNote: "Drift has official human backend API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "freshworks-crm",
		SpecRefID:  "freshworks-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.freshworks.com/crm/api/"},
		SourceNote: "Freshworks CRM has official human REST API docs but no recorded stable public official OpenAPI document; Authorization token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "getresponse",
		SpecRefID:  "getresponse-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://apidocs.getresponse.com/v3/authentication", "https://apidocs.getresponse.com/v3"},
		SourceNote: "GetResponse has official human API docs but no recorded stable public official OpenAPI document; API-key and OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "highlevel",
		SpecRefID:  "highlevel-contacts-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://github.com/GoHighLevel/highlevel-api-docs", "https://marketplace.gohighlevel.com/docs/", "https://help.gohighlevel.com/support/solutions/articles/48001060529-highlevel-api-documentation"},
		SourceNote: "HighLevel's official API v2 OpenAPI modules include bearer security schemes and operation-level security metadata for agency and location access.",
	},
	{
		ProviderID: "keap",
		SpecRefID:  "keap-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.keap.com/docs/rest/1000/"},
		SourceNote: "Keap has official human REST API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mailerlite",
		SpecRefID:  "mailerlite-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.mailerlite.com/docs/", "https://developers-classic.mailerlite.com/docs/authentication", "https://www.mailerlite.com/help/where-to-find-the-mailerlite-api-key-groupid-and-documentation"},
		SourceNote: "MailerLite has official human API docs but no recorded stable public official OpenAPI document; current bearer-token and Classic API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mautic",
		SpecRefID:  "mautic-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.mautic.org/", "https://kb.mautic.org/article/what-is-mautic-039%3Bs-api.html"},
		SourceNote: "Mautic has official human API docs but no recorded stable public official OpenAPI document; OAuth and Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "monica-crm",
		SpecRefID:  "monica-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.monicahq.com/api"},
		SourceNote: "Monica has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "salesmate",
		SpecRefID:  "salesmate-api-auth-example",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://support.salesmate.io/hc/en-us/articles/12864176787609-What-Are-API-s-Usage-and-Working", "https://support.salesmate.io/hc/en-us/articles/360043653751-Auto-enroll-Contacts-to-sequence-whenever-a-Deal-stage-is-updated-via-Webhooks", "https://apidocs.salesmate.io/"},
		SourceNote: "Salesmate has official human API docs but no recorded stable public official OpenAPI document; sessionToken and x-linkname metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "sap-s4hana",
		SpecRefID:  "sap-s4hana-cloud-api-hub-docs",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://help.sap.com/docs/SAP_S4HANA_CLOUD/0f69f8fb28ac4bf48d2b57b9637e81fa/1e60f14bdc224c2c975c8fa8bcfd7f3f.html?locale=en-US", "https://developers.sap.com/tutorials/hcp-abh-getting-started..html", "https://userapps.support.sap.com/sap/support/knowledge/en/3582906"},
		SourceNote: "SAP S/4HANA APIs span Business Accelerator Hub OpenAPI/OData/SOAP source families and tenant/configuration-specific auth contexts; apitools must not hide OData or SOAP semantics in a generic overlay or resolve SAP credentials.",
	},
	{
		ProviderID: "sap-successfactors",
		SpecRefID:  "sap-successfactors-available-apis",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://userapps.support.sap.com/sap/support/knowledge/en/2613670", "https://help.sap.com/docs/successfactors-platform/sap-successfactors-api-reference-guide-odata-v2/authentication?locale=en-US", "https://userapps.support.sap.com/sap/support/knowledge/en/3641488"},
		SourceNote: "SAP SuccessFactors APIs are primarily OData and SOAP families with tenant/module-specific authentication; apitools records source metadata only and does not implement OData lowering or resolve SuccessFactors credentials in M55.",
	},
	{
		ProviderID: "sendy",
		SpecRefID:  "sendy-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://sendy.co/api"},
		SourceNote: "Sendy has official human API docs but no recorded stable public official OpenAPI document; api_key form/query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "vero",
		SpecRefID:  "vero-track-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.getvero.com/api-reference/track/overview", "https://help.getvero.com/developer-docs/overview"},
		SourceNote: "Vero has official human API docs but no recorded stable public official OpenAPI document; auth_token request-parameter metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "jotform",
		SpecRefID:  "jotform-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.jotform.com/docs/", "https://api.jotform.com/"},
		SourceNote: "JotForm has official human API docs but no recorded stable public official OpenAPI document; APIKEY metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "formio",
		SpecRefID:  "formio-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.form.io/developers/authentication", "https://help.form.io/developers/introduction/api-documentation"},
		SourceNote: "Form.io has official human API docs but no recorded stable public official OpenAPI document; token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "formstack",
		SpecRefID:  "formstack-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.formstack.com/reference/authorization", "https://developers.formstack.com/v2.0/reference/api-overview"},
		SourceNote: "Formstack has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "surveymonkey",
		SpecRefID:  "surveymonkey-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.surveymonkey.com/v3/docs", "https://help.surveymonkey.com/en/surveymonkey/integrations/surveymonkey-api/"},
		SourceNote: "SurveyMonkey has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "kobotoolbox",
		SpecRefID:  "kobotoolbox-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://kf.kobotoolbox.org/api/v2/schema/", "https://kf.kobotoolbox.org/api/v2/docs/"},
		SourceNote: "KoBoToolbox official API v2 OpenAPI schema declares TokenAuth, BasicAuth, and SCIM bearer security metadata with operation-level requirements.",
	},
	{
		ProviderID: "bubble",
		SpecRefID:  "bubble-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://manual.bubble.io/help-guides/integrations/api/the-bubble-api", "https://manual.bubble.io/core-resources/api/data-api"},
		SourceNote: "Bubble has official human docs and app-specific Swagger metadata but no stable provider-wide public official OpenAPI document; bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cal",
		SpecRefID:  "cal-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://cal.com/docs", "https://cal.com/docs/api-reference/v2/openapi.json"},
		SourceNote: "Cal.com official API v2 OpenAPI metadata requires Authorization headers but does not declare reusable OpenAPI security schemes; bearer metadata comes from a review overlay.",
	},
	{
		ProviderID: "cockpit",
		SpecRefID:  "cockpit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://getcockpit.com/documentation/core/api", "https://getcockpit.com/documentation/core"},
		SourceNote: "Cockpit has official human API docs but no recorded stable public official OpenAPI document; token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "stackby",
		SpecRefID:  "stackby-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.stackby.com/en/articles/29-developer-api", "https://help.stackby.com/en/articles/124-how-to-get-your-api-key-in-stackby"},
		SourceNote: "Stackby has official human API docs but no recorded stable public official OpenAPI document; api-key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "airtop",
		SpecRefID:  "airtop-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.airtop.ai/openapi.json", "https://docs.airtop.ai/api-reference/airtop-api"},
		SourceNote: "Airtop's official OpenAPI document declares bearer metadata and repeated Authorization header parameters; bearer metadata is carried as a present-incomplete review overlay.",
	},
	{
		ProviderID: "filemaker",
		SpecRefID:  "filemaker-data-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.claris.com/en/data-api-guide/content/data-api-reference.html", "https://help.claris.com/en/data-api-guide/"},
		SourceNote: "FileMaker has official human Data API docs but no recorded stable public official OpenAPI document; Basic login and bearer session metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "facebook",
		SpecRefID:  "facebook-graph-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/graph-api/", "https://developers.facebook.com/docs/pages-api/"},
		SourceNote: "Facebook has official Graph API human docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "facebook-lead-ads",
		SpecRefID:  "facebook-lead-ads-retrieval-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/marketing-api/guides/lead-ads/retrieving/", "https://developers.facebook.com/docs/marketing-api/guides/lead-ads/webhooks/"},
		SourceNote: "Facebook Lead Ads has official human docs but no recorded stable public official OpenAPI document; access-token and app-review-sensitive permission metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "linkedin",
		SpecRefID:  "linkedin-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://learn.microsoft.com/en-us/linkedin/marketing/", "https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow"},
		SourceNote: "LinkedIn has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata and version/header requirements come from advisory overlay notes.",
	},
	{
		ProviderID: "line",
		SpecRefID:  "line-messaging-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/line/line-openapi/main/messaging-api.yml", "https://github.com/line/line-openapi"},
		SourceNote: "LINE's official Messaging API OpenAPI document includes bearer channel access-token security metadata with root security requirements.",
	},
	{
		ProviderID: "mattermost",
		SpecRefID:  "mattermost-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.mattermost.com/mattermost-openapi-v4.yaml", "https://developers.mattermost.com/contribute/more-info/server/rest-api/"},
		SourceNote: "Mattermost's official API v4 OpenAPI document includes bearer security scheme metadata and operation-level security requirements.",
	},
	{
		ProviderID: "messagebird",
		SpecRefID:  "messagebird-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.messagebird.com/api/"},
		SourceNote: "MessageBird has official human API docs but no recorded stable public official OpenAPI document; Authorization: AccessKey metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "matrix",
		SpecRefID:  "matrix-client-server-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://spec.matrix.org/latest/client-server-api/api.json", "https://spec.matrix.org/latest/client-server-api/"},
		SourceNote: "Matrix's official Client-Server API OpenAPI document includes bearer and deprecated query access-token security schemes with operation-level requirements.",
	},
	{
		ProviderID: "rocket-chat",
		SpecRefID:  "rocket-chat-messaging-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/authentication.yaml", "https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/messaging.yaml", "https://developer.rocket.chat/apidocs"},
		SourceNote: "Rocket.Chat's official OpenAPI modules model x-Auth-Token and x-User-Id as header parameters rather than reusable security schemes; combined header requirements come from advisory overlay notes.",
	},
	{
		ProviderID: "twake",
		SpecRefID:  "twake-developers-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://doc.twake.app/developers-api/api-reference/auth", "https://doc.twake.app/developers-api/api-reference/message/post-request"},
		SourceNote: "Twake has official Developers API human docs but no recorded stable public official OpenAPI document; Basic application authentication metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "twist",
		SpecRefID:  "twist-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.twist.com/v3/"},
		SourceNote: "Twist has official API v3 human docs but no recorded stable public official OpenAPI document; OAuth/test bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "twitter",
		SpecRefID:  "twitter-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.twitter.com/2/openapi.json", "https://docs.x.com/x-api"},
		SourceNote: "X/Twitter's official API v2 OpenAPI document includes bearer token, OAuth 2.0 authorization code, and OAuth 1.0a-style security metadata with operation-level requirements.",
	},
	{
		ProviderID: "whatsapp",
		SpecRefID:  "whatsapp-cloud-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/whatsapp/cloud-api/", "https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages"},
		SourceNote: "WhatsApp Cloud API has official human docs but no recorded stable public official OpenAPI document; Meta Graph API bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "magento",
		SpecRefID:  "magento-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.adobe.com/commerce/webapi/rest/reference/", "https://developer.adobe.com/commerce/webapi/get-started/authentication/"},
		SourceNote: "Adobe Commerce/Magento has official human REST docs and instance-local Swagger generation but no recorded stable public official OpenAPI document; bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "paddle",
		SpecRefID:  "paddle-api-v1-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/PaddleHQ/paddle-openapi/main/v1/openapi.yaml", "https://github.com/PaddleHQ/paddle-openapi", "https://developer.paddle.com/api-reference/about/authentication"},
		SourceNote: "Paddle's official OpenAPI document includes bearer API-key security metadata for the Paddle Billing API.",
	},
	{
		ProviderID: "gumroad",
		SpecRefID:  "gumroad-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://gumroad.com/api", "https://gumroad.com/oauth/applications"},
		SourceNote: "Gumroad has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "invoice-ninja",
		SpecRefID:  "invoice-ninja-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api-docs.invoicing.co/", "https://invoiceninja.github.io/docs/developer-guide"},
		SourceNote: "Invoice Ninja has official OpenAPI-rendered human API docs but no recorded stable standalone downloadable OpenAPI document; X-API-TOKEN metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "odoo",
		SpecRefID:  "odoo-external-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.odoo.com/documentation/18.0/developer/reference/external_api.html"},
		SourceNote: "Odoo has official external API human docs but no recorded stable public official OpenAPI document; API-key/password RPC authentication metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "erpnext",
		SpecRefID:  "erpnext-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.frappe.io/erpnext/rest-api", "https://docs.frappe.io/framework/user/en/api/rest#1-token-based-authentication"},
		SourceNote: "ERPNext/Frappe has official human REST API docs but no recorded stable public official OpenAPI document; token authentication metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "woocommerce",
		SpecRefID:  "woocommerce-rest-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://woocommerce.github.io/woocommerce-rest-api-docs/#authentication", "https://developer.woocommerce.com/docs/apis/rest-api/v3/"},
		SourceNote: "WooCommerce has official human REST API docs but no recorded stable public official OpenAPI document; Basic and query consumer-key auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "wise",
		SpecRefID:  "wise-platform-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.wise.com/api-reference", "https://docs.wise.com/api-docs/features"},
		SourceNote: "Wise has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "dhl",
		SpecRefID:  "dhl-shipment-tracking-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.dhl.com/api-reference/shipment-tracking?language_content_entity=en"},
		SourceNote: "DHL has official Shipment Tracking human API docs but no recorded stable public downloadable official OpenAPI document; subscription-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "onfleet",
		SpecRefID:  "onfleet-api-reference",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.onfleet.com/reference", "https://support.onfleet.com/hc/en-us/articles/360045763292-API"},
		SourceNote: "Onfleet has official human API docs but no recorded stable public official OpenAPI document; Basic API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "unleashed-software",
		SpecRefID:  "unleashed-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://apidocs.unleashedsoftware.com/", "https://apidocs.unleashedsoftware.com/Authentication"},
		SourceNote: "Unleashed has official human API docs but no recorded stable public official OpenAPI document; API ID, signature, and client-type metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "workable",
		SpecRefID:  "workable-api-reference",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://workable.readme.io/reference", "https://help.workable.com/hc/en-us/articles/115013356548-Workable-API-Documentation"},
		SourceNote: "Workable has official human API docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "workday",
		SpecRefID:  "workday-soap-api-reference",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://community-content.workday.com/en-us/public/products/platform-and-product-extensions/soap-api-reference.html", "https://community.workday.com/sites/default/files/file-hosting/productionapi/index.html", "https://community.workday.com/sites/default/files/file-hosting/restapi/index.html"},
		SourceNote: "Workday API coverage spans WWS SOAP/WSDL and REST documentation, with tenant/security configuration outside public catalog metadata; apitools must not add WSDL/SOAP parsing or contact Workday tenants in M55.",
	},
	{
		ProviderID: "bitwarden",
		SpecRefID:  "bitwarden-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://bitwarden.com/help/api/"},
		SourceNote: "Bitwarden has official Public API docs but no recorded stable public downloadable official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cisco-webex",
		SpecRefID:  "cisco-webex-rooms-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.webex.com/docs/api/v1/rooms", "https://developer.webex.com/docs/integrations"},
		SourceNote: "Cisco Webex has official human API docs but no recorded stable public downloadable official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cortex",
		SpecRefID:  "cortex-api-guide",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.strangebee.com/cortex/api/api-guide/"},
		SourceNote: "Cortex has official human API docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "home-assistant",
		SpecRefID:  "home-assistant-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.home-assistant.io/docs/api/rest/"},
		SourceNote: "Home Assistant has official REST API human docs but no recorded stable public official OpenAPI document; long-lived bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "misp",
		SpecRefID:  "misp-automation-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/MISP/MISP/2.4/app/webroot/doc/openapi.yaml", "https://www.misp-project.org/openapi/"},
		SourceNote: "MISP's official Automation API OpenAPI document includes an Authorization header API-key security scheme and root security requirement.",
	},
	{
		ProviderID: "netscaler",
		SpecRefID:  "netscaler-adc-nitro-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer-docs.netscaler.com/en-us/adc-nitro-api/current-release/api-reference.html"},
		SourceNote: "NetScaler ADC has official NITRO API human docs but no recorded stable public official OpenAPI document; NITRO session cookie metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "rundeck",
		SpecRefID:  "rundeck-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.rundeck.com/docs/files/rundeck-api.yml", "https://docs.rundeck.com/docs/api/"},
		SourceNote: "Rundeck's official OpenAPI document includes API-token, password/session, JWT, and webhook-token security schemes with root security requirements.",
	},
	{
		ProviderID: "securityscorecard",
		SpecRefID:  "securityscorecard-api-docs-swagger-v2",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.securityscorecard.io/api-docs"},
		SourceNote: "SecurityScorecard's official Swagger document includes Authorization header Token security definitions and root security requirements.",
	},
	{
		ProviderID: "urlscan",
		SpecRefID:  "urlscan-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.urlscan.io/_bundle/apis/urlscan-openapi.json?download", "https://docs.urlscan.io/apis/urlscan-openapi"},
		SourceNote: "urlscan.io's official OpenAPI document includes api-key header security metadata and root security requirements.",
	},
	{
		ProviderID: "venafi",
		SpecRefID:  "venafi-tls-protect-cloud-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.venafi.com/tlsprotectcloud/reference", "https://developer.venafi.com/tlsprotectdatacenter/reference"},
		SourceNote: "Venafi has official human API docs for TLS Protect Cloud and Datacenter but no recorded stable public official OpenAPI document; API-key and bearer-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "wekan",
		SpecRefID:  "wekan-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/wekan/wekan/wiki/REST-API", "https://github.com/wekan/wekan/wiki/REST-API-Authentication"},
		SourceNote: "Wekan has official REST API wiki docs and OpenAPI generation tooling but no recorded stable public generated OpenAPI artifact; bearer session-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "zammad",
		SpecRefID:  "zammad-api-intro-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.zammad.org/en/latest/api/intro.html"},
		SourceNote: "Zammad has official human API docs but no recorded stable public official OpenAPI document; Basic and token authentication metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "humantic-ai",
		SpecRefID:  "humantic-ai-api-root",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.humantic.ai/"},
		SourceNote: "Humantic AI has official human API docs but no recorded stable public official OpenAPI document; apikey query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "hunter",
		SpecRefID:  "hunter-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://hunter.io/api-documentation"},
		SourceNote: "Hunter has official human API docs but no recorded stable public official OpenAPI document; api_key query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "jina-ai",
		SpecRefID:  "jina-ai-search-foundation-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.jina.ai/openapi.json", "https://r.jina.ai/openapi.json", "https://s.jina.ai/openapi.json", "https://jina.ai/en-US/reader/"},
		SourceNote: "Jina AI publishes official OpenAPI documents, but Reader/Search specs omit reusable security schemes and DeepSearch probes require authorization; bearer API-key metadata is carried as catalog overlay guidance.",
	},
	{
		ProviderID: "lingvanex",
		SpecRefID:  "lingvanex-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.lingvanex.com/reference/user-guide", "https://github.com/lingvanex-mt/python-translation-api"},
		SourceNote: "LingvaNex has official human API docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mistral-ai",
		SpecRefID:  "mistral-api-specs-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.mistral.ai/api", "https://docs.mistral.ai/admin/security-access/api-keys"},
		SourceNote: "Mistral AI has official rendered API specs but no recorded stable public downloadable official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "openai",
		SpecRefID:  "openai-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://app.stainless.com/api/spec/documented/openai/openapi.documented.yml", "https://github.com/openai/openai-openapi", "https://platform.openai.com/docs/api-reference"},
		SourceNote: "OpenAI's official documented OpenAPI document includes ApiKeyAuth bearer security and a root security requirement.",
	},
	{
		ProviderID: "perplexity",
		SpecRefID:  "perplexity-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.perplexity.ai/openapi.json", "https://docs.perplexity.ai/api-reference/chat-completions-post"},
		SourceNote: "Perplexity's official OpenAPI document includes an HTTP bearer security scheme and operation-level security requirements.",
	},
	{
		ProviderID: "mindee",
		SpecRefID:  "mindee-api-overview-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.mindee.com/integrations/api-overview", "https://docs.mindee.com/integrations/api-keys"},
		SourceNote: "Mindee has official human API docs but no recorded stable public official OpenAPI document; Token and X-Inferuser-Token header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "peekalink",
		SpecRefID:  "peekalink-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.peekalink.io/mintlify", "https://docs.peekalink.io/api-reference/link-preview/get-a-link-preview"},
		SourceNote: "Peekalink's official OpenAPI document includes a bearer security scheme and operation-level security requirement.",
	},
	{
		ProviderID: "phantombuster",
		SpecRefID:  "phantombuster-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://hub.phantombuster.com/docs/api", "https://hub.phantombuster.com/reference"},
		SourceNote: "Phantombuster has official human API docs but no recorded stable public official OpenAPI document; X-Phantombuster-Key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "uplead",
		SpecRefID:  "uplead-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.uplead.com/"},
		SourceNote: "UpLead has official human API docs but no recorded stable public official OpenAPI document; Authorization header API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "dropcontact",
		SpecRefID:  "dropcontact-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.dropcontact.com/"},
		SourceNote: "Dropcontact has official human API docs but no recorded stable public official OpenAPI document; X-Access-Token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mocean",
		SpecRefID:  "mocean-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://moceanapi.com/docs"},
		SourceNote: "Mocean has official human API docs but no recorded stable public official OpenAPI document; mocean-api-key and mocean-api-secret request-parameter metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "msg91",
		SpecRefID:  "msg91-sms-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.msg91.com/sms", "https://docs.msg91.com/overview"},
		SourceNote: "MSG91 has official human API docs but no recorded stable public official OpenAPI document; authkey request-parameter metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "plivo",
		SpecRefID:  "plivo-messaging-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.plivo.com/docs/messaging/api/message/retrieve-a-message", "https://www.plivo.com/docs/voice/api/call/the-call-object/"},
		SourceNote: "Plivo has official human Messaging and Voice API docs but no recorded stable public official OpenAPI document; Auth ID/Auth Token HTTP Basic metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "pushbullet",
		SpecRefID:  "pushbullet-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.pushbullet.com/"},
		SourceNote: "Pushbullet has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "pushcut",
		SpecRefID:  "pushcut-integrations-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.pushcut.io/support/integrations", "https://www.pushcut.io/support/notifications"},
		SourceNote: "Pushcut has official human API and notification docs but no recorded stable public official OpenAPI document; API-Key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "pushover",
		SpecRefID:  "pushover-message-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://pushover.net/api", "https://pushover.net/api/client"},
		SourceNote: "Pushover has official human Message and Client API docs but no recorded stable public official OpenAPI document; app token and user/group key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "signl4",
		SpecRefID:  "signl4-api-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://connect.signl4.com/api/docs/v1/swagger.json", "https://connect.signl4.com/api/docs/index.html"},
		SourceNote: "SIGNL4's official OpenAPI document declares API-key and OAuth schemes but leaves root security anonymous in the reviewed artifact; X-S4-Api-Key metadata is carried as catalog overlay guidance.",
	},
	{
		ProviderID: "sms77",
		SpecRefID:  "seven-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.seven.io/en/rest-api/authentication", "https://docs.seven.io/en/rest-api/endpoints/sms"},
		SourceNote: "seven.io/sms77 has official human API docs but no recorded stable public official OpenAPI document; X-Api-Key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "vonage",
		SpecRefID:  "vonage-messages-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developer.vonage.com/api/v1/developer/api/file/messages?format=yml&vendorId=vonage", "https://developer.vonage.com/en/api/messages"},
		SourceNote: "Vonage's official Messages API OpenAPI document includes bearer and basic security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "zulip",
		SpecRefID:  "zulip-rest-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zulip/zulip/main/zerver/openapi/zulip.yaml", "https://docs.zulip.com/api/rest"},
		SourceNote: "Zulip's official REST API OpenAPI document includes HTTP Basic security metadata with root and operation security requirements.",
	},
	{
		ProviderID: "gotify",
		SpecRefID:  "gotify-server-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/gotify/server/master/docs/spec.json", "https://gotify.net/docs/index"},
		SourceNote: "Gotify's official Swagger document includes application, client, admin, and stream token security definitions with operation-level security requirements.",
	},
	{
		ProviderID: "emelia",
		SpecRefID:  "emelia-graphql-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs-old.emelia.io/"},
		SourceNote: "Emelia has official GraphQL human API docs but no recorded stable public official OpenAPI document; Authorization header API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "disqus",
		SpecRefID:  "disqus-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://disqus.com/api/docs/auth/", "https://disqus.com/api/docs/"},
		SourceNote: "Disqus has official human API docs but no recorded stable public official OpenAPI document; access_token and api_key request-parameter metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "medium",
		SpecRefID:  "medium-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/Medium/medium-api-docs"},
		SourceNote: "Medium has official human API docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "npm",
		SpecRefID:  "npm-registry-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md", "https://docs.npmjs.com/cli/v11/using-npm/registry"},
		SourceNote: "npm has official registry human docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "spotify",
		SpecRefID:  "spotify-web-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developer.spotify.com/reference/web-api/open-api-schema.yaml", "https://developer.spotify.com/documentation/web-api/concepts/authorization"},
		SourceNote: "Spotify's official Web API OpenAPI document includes OAuth2 authorization-code security metadata with operation-level security requirements.",
	},
	{
		ProviderID: "storyblok",
		SpecRefID:  "storyblok-content-delivery-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.storyblok.com/docs/api/content-delivery/v2", "https://www.storyblok.com/docs/api/management/getting-started/authentication/"},
		SourceNote: "Storyblok has official Content Delivery and Management API docs but no recorded stable public official OpenAPI document; content-token and management bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "strapi",
		SpecRefID:  "strapi-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.strapi.io/cms/api/rest", "https://docs.strapi.io/cms/plugins/documentation"},
		SourceNote: "Strapi has official REST API docs and app-local OpenAPI generation guidance but no recorded stable provider-wide public official OpenAPI document; bearer JWT/API-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "wordpress",
		SpecRefID:  "wordpress-rest-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.wordpress.org/rest-api/using-the-rest-api/authentication/", "https://developer.wordpress.org/rest-api/reference/"},
		SourceNote: "WordPress has official REST API docs but no stable provider-wide public official OpenAPI document; Basic/application-password and WordPress.com OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "wufoo",
		SpecRefID:  "wufoo-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.wufoo.com/docs/api/v3/", "https://www.wufoo.com/docs/api/v3/forms/"},
		SourceNote: "Wufoo has official API v3 human docs but no recorded stable public official OpenAPI document; API-key Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "yourls",
		SpecRefID:  "yourls-passwordless-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://yourls.org/docs/guide/advanced/api", "https://yourls.org/docs/guide/advanced/passwordless-api"},
		SourceNote: "YOURLS has official human API docs but no recorded stable public official OpenAPI document; signature query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "raindrop",
		SpecRefID:  "raindrop-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.raindrop.io/v1/authentication", "https://developer.raindrop.io/v1/"},
		SourceNote: "Raindrop.io has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "toggl",
		SpecRefID:  "toggl-track-api-swagger",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://engineering.toggl.com/assets/files/api-e95bb70f0fdd53fbd754f1dcf1041a9f.json", "https://engineering.toggl.com/docs/track/authentication/"},
		SourceNote: "Toggl's official Swagger document defines BasicAuth and uses operation security metadata, but also references an undeclared OAuth2 scheme in some operations; the reviewed auth state remains present-incomplete.",
	},
	{
		ProviderID: "travis-ci",
		SpecRefID:  "travis-ci-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.travis-ci.com/authentication", "https://developer.travis-ci.com/"},
		SourceNote: "Travis CI has official human API v3 docs but no recorded stable public official OpenAPI document; token Authorization and Travis-API-Version header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "flow",
		SpecRefID:  "flow-api-overview",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.getflow.com/", "https://developer.getflow.com/errors"},
		SourceNote: "Flow has official human API docs but no recorded stable public official OpenAPI document; bearer personal-access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "gotowebinar",
		SpecRefID:  "goto-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.goto.com/GoToWebinarV2/", "https://developer.goto.com/guides/Authentication/"},
		SourceNote: "GoTo Webinar has official human API docs but no recorded stable public official OpenAPI document; OAuth2 bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "halopsa",
		SpecRefID:  "halopsa-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://apidoc.halopsa.com/", "https://support.halopsa.com/portal/kb/articles/haloapi"},
		SourceNote: "HaloPSA has official human API docs and tenant-specific API hosts but no recorded stable public provider-wide OpenAPI document; OAuth2 client-credentials metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "lonescale",
		SpecRefID:  "lonescale-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help-center.lonescale.com/en/articles/6454360-lonescale-public-api", "https://public-api.lonescale.com/api"},
		SourceNote: "LoneScale has official human public API docs but no recorded stable public official OpenAPI document; X-API-KEY metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "lemlist",
		SpecRefID:  "lemlist-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.lemlist.com/api-reference/getting-started/authentication", "https://developer.lemlist.com/api-reference/getting-started/overview"},
		SourceNote: "lemlist has official human API docs but no recorded stable public official OpenAPI document; Basic auth with empty username and API key password comes from advisory overlay notes.",
	},
	{
		ProviderID: "orbit",
		SpecRefID:  "orbit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.orbit.love/reference", "https://orbit.love/blog/orbit-is-joining-postman"},
		SourceNote: "Orbit's official docs endpoint was not reliably reachable during M33 review and the product has shut down; bearer token metadata is retained only as historical advisory metadata.",
	},
	{
		ProviderID: "oura",
		SpecRefID:  "oura-api-v2-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://cloud.ouraring.com/v2/static/json/openapi-1.29.json", "https://cloud.ouraring.com/v2/docs"},
		SourceNote: "Oura's official OpenAPI document references BearerAuth in operation security examples but omits the matching components.securitySchemes entry; a catalog overlay records the missing bearer scheme metadata.",
	},
	{
		ProviderID: "profitwell",
		SpecRefID:  "profitwell-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://profitwellapiv2.docs.apiary.io/"},
		SourceNote: "ProfitWell has official human API docs but no recorded stable public official OpenAPI document; direct Authorization header token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "quickbase",
		SpecRefID:  "quickbase-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.quickbase.com/", "https://help.quickbase.com/docs/authentication-and-secure-access", "https://help.quickbase.com/docs/create-and-use-user-tokens"},
		SourceNote: "Quickbase has official REST API human docs but no recorded stable public official OpenAPI document; bearer user-token and realm-hostname header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "syncromsp",
		SpecRefID:  "syncromsp-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api-docs.syncromsp.com/swagger.json", "https://api-docs.syncromsp.com/"},
		SourceNote: "SyncroMSP's official OpenAPI document includes an Authorization header API-key security scheme and server variable metadata.",
	},
	{
		ProviderID: "taiga",
		SpecRefID:  "taiga-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.taiga.io/api.html", "https://docs.taiga.io/api.html#_authentication"},
		SourceNote: "Taiga has official human REST API docs but no recorded stable public official OpenAPI document; bearer auth-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "tapfiliate",
		SpecRefID:  "tapfiliate-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://tapfiliate.com/docs/rest/", "https://tapfiliate.com/docs/rest/#authentication"},
		SourceNote: "Tapfiliate has official human REST API docs but no recorded stable public official OpenAPI document; Api-Key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "thehive",
		SpecRefID:  "thehive-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/TheHive-Project/api-docs/master/thehive.yaml", "https://github.com/TheHive-Project/api-docs"},
		SourceNote: "TheHive Project's official TheHive 4 OpenAPI document includes an Authenticated HTTP bearer security scheme.",
	},
	{
		ProviderID: "thehive-project",
		SpecRefID:  "thehive-project-api-v5-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.strangebee.com/thehive/api-docs/docs.yaml", "https://docs.strangebee.com/thehive/api-docs/"},
		SourceNote: "TheHive 5 official OpenAPI document includes bearer API-key, Basic, session cookie, and header-based security schemes.",
	},
	{
		ProviderID: "apitemplate-io",
		SpecRefID:  "apitemplate-io-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.apitemplate.io/api/index.html"},
		SourceNote: "APITemplate.io has official human API docs but no recorded stable public official OpenAPI document; X-API-KEY metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "currents",
		SpecRefID:  "currents-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.currents.dev/resources/api/authentication", "https://docs.currents.dev/api"},
		SourceNote: "Currents has official human API docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "demio",
		SpecRefID:  "demio-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://publicdemioapi.docs.apiary.io/", "https://help.demio.com/en/articles/4544025-api-limitations"},
		SourceNote: "Demio has official human API docs but no recorded stable public official OpenAPI document; paired Api-Key and Api-Secret header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "egoi",
		SpecRefID:  "egoi-marketing-api-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.egoiapp.com/openapi", "https://developers.e-goi.com/api/v3/"},
		SourceNote: "E-goi's official Marketing API v3 OpenAPI document includes an Apikey header security scheme and operation security requirements.",
	},
	{
		ProviderID: "mailcheck",
		SpecRefID:  "mailcheck-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://app.mailcheck.co/openapi.json", "https://app.mailcheck.co/docs?from=docs#post-/v1/singleEmail-check"},
		SourceNote: "Mailcheck's official OpenAPI document includes a bearer API-key security scheme.",
	},
	{
		ProviderID: "mandrill",
		SpecRefID:  "mandrill-transactional-fundamentals",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://mailchimp.com/developer/transactional/docs/fundamentals/", "https://mailchimp.com/developer/transactional/api/"},
		SourceNote: "Mailchimp Transactional has official human API docs but no recorded stable public official OpenAPI document; the required key payload field is represented as advisory credential-placement metadata.",
	},
	{
		ProviderID: "metabase",
		SpecRefID:  "metabase-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.metabase.com/docs/latest/api", "https://www.metabase.com/learn/metabase-basics/administration/administration-and-operation/metabase-api"},
		SourceNote: "Metabase exposes human API docs and instance-specific live OpenAPI docs, but no recorded stable provider-hosted OpenAPI artifact; session-header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "nextcloud",
		SpecRefID:  "nextcloud-server-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.nextcloud.com/server/latest/developer_manual/_static/openapi.json", "https://docs.nextcloud.com/server/latest/developer_manual/digging_deeper/apis.html"},
		SourceNote: "Nextcloud's official server OpenAPI document includes Basic and bearer security schemes for documented server API endpoints.",
	},
	{
		ProviderID: "nvidia-dsx-air",
		SpecRefID:  "nvidia-dsx-air-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://dsx-air.nvidia.com/api/schema/", "https://docs.nvidia.com/networking-ethernet-software/nvidia-air-v2/Authentication/"},
		SourceNote: "NVIDIA DSX Air publishes an official OpenAPI schema, but the schema currently omits reusable security scheme metadata; NGC API-key bearer authentication is represented as catalog overlay guidance.",
	},
	{
		ProviderID: "philips-hue",
		SpecRefID:  "philips-hue-api-v2-notice",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.meethue.com/new-hue-api/", "https://developers.meethue.com/terms-of-use-and-conditions/"},
		SourceNote: "Philips Hue official API docs are login-gated and no stable public official OpenAPI document was recorded; OAuth2 remote API metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "postbin",
		SpecRefID:  "postbin-service-page",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://www.postb.in/"},
		SourceNote: "PostBin's public request-bin utility surface is anonymous in the reviewed scope; no credential-bearing API docs or official OpenAPI document were found during M34 review.",
	},
	{
		ProviderID: "segment",
		SpecRefID:  "segment-public-api-docs",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.segmentapis.build/", "https://docs.segmentapis.build/tag/Authentication"},
		SourceNote: "Segment's official Public API docs are rendered from an embedded OpenAPI 3.0.3 definition with bearer token security, and the authentication docs require Authorization bearer API tokens over HTTPS.",
	},
	{
		ProviderID: "strava",
		SpecRefID:  "strava-api-v3-swagger",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.strava.com/swagger/swagger.json", "https://developers.strava.com/docs/authentication/", "https://developers.strava.com/docs/reference/"},
		SourceNote: "Strava's official Swagger document includes an OAuth2 access-code security definition and root security requirement; the official reference states API requests require authentication.",
	},
	{
		ProviderID: "zoho",
		SpecRefID:  "zoho-crm-v8-records-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://github.com/zoho/crm-oas", "https://raw.githubusercontent.com/zoho/crm-oas/main/v8.0/record.json", "https://www.zoho.com/crm/developer/docs/api/v8/oauth-overview.html"},
		SourceNote: "Zoho CRM's official v8.0 OpenAPI documents include OAuth2 authorization-code security schemes and operation security requirements for protected CRM APIs.",
	},
	{
		ProviderID: "uproc",
		SpecRefID:  "uproc-public-profile-tool-docs",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://uproc.io/", "https://uproc.io/blog/how-to-get-public-profile-by-sales-profile", "https://uproc.io/blog/how-to-get-parsed-and-validated-phone"},
		SourceNote: "uProc official service and tool pages reference API use and email/API-key authentication, but M35 review did not find stable official text describing an OpenAPI security scheme or credential transport.",
	},
}

func validateSecurityClassification(classification SecurityClassification, providers map[string]Provider) error {
	if strings.TrimSpace(classification.ProviderID) == "" {
		return fmt.Errorf("missing provider id")
	}
	if !validID(classification.ProviderID) {
		return fmt.Errorf("invalid provider id %q", classification.ProviderID)
	}
	provider, ok := providers[classification.ProviderID]
	if !ok {
		return fmt.Errorf("unknown provider %q", classification.ProviderID)
	}
	if strings.TrimSpace(classification.SpecRefID) != "" {
		if !validID(classification.SpecRefID) {
			return fmt.Errorf("invalid spec ref id %q", classification.SpecRefID)
		}
		if !providerHasSpecRef(provider, classification.SpecRefID) {
			return fmt.Errorf("unknown spec ref %q for provider %q", classification.SpecRefID, classification.ProviderID)
		}
	}
	if !validAuthCompletenessStatus(classification.Status) {
		return fmt.Errorf("invalid status %q", classification.Status)
	}
	if strings.TrimSpace(classification.SourceNote) == "" {
		return fmt.Errorf("missing source note")
	}
	if len(classification.SourceRefs) == 0 {
		return fmt.Errorf("missing source refs")
	}
	for i, ref := range classification.SourceRefs {
		if !validHTTPSURL(ref) {
			return fmt.Errorf("source ref[%d]: must be https", i)
		}
	}
	return validateUniqueStrings("source_ref", classification.SourceRefs)
}

func sortSecurityClassifications(classifications []SecurityClassification) {
	sort.SliceStable(classifications, func(i, j int) bool {
		return classifications[i].ProviderID < classifications[j].ProviderID
	})
}

func cloneSecurityClassifications(in []SecurityClassification) []SecurityClassification {
	out := make([]SecurityClassification, len(in))
	for i, classification := range in {
		out[i] = classification
		out[i].SourceRefs = append([]string(nil), classification.SourceRefs...)
	}
	return out
}

func appendIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, value)
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
