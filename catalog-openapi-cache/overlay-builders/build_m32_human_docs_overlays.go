//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"sort"
)

type overlaySpec struct {
	ProviderID  string
	Title       string
	Description string
	ServerURL   string
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		disqusOverlay(),
		mediumOverlay(),
		npmOverlay(),
		storyblokOverlay(),
		strapiOverlay(),
		wordpressOverlay(),
		wufooOverlay(),
		yourlsOverlay(),
		raindropOverlay(),
		travisCIOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func disqusOverlay() overlaySpec {
	security := map[string]map[string]any{
		"disqusAccessToken": {"type": "apiKey", "in": "query", "name": "access_token", "description": "Disqus access token carried in the access_token request parameter."},
		"disqusAPIKey":      {"type": "apiKey", "in": "query", "name": "api_key", "description": "Disqus public API key carried in the api_key request parameter."},
	}
	return overlaySpec{
		ProviderID:  "disqus",
		Title:       "Disqus API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Disqus human API documentation. This is not an official Disqus OpenAPI document.",
		ServerURL:   "https://disqus.com/api/3.0",
		Sources:     []string{"https://disqus.com/api/docs/", "https://disqus.com/api/docs/auth/"},
		SourceNote:  "Disqus publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected forum, thread, and post endpoints.",
		Security:    security,
		Schemas:     []string{"DisqusObject", "DisqusCollection", "DisqusError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/disqus-api-overlay.json",
		Paths: map[string]map[string]any{
			"/forums/details.json":        {"get": op("getDisqusForum", "Get forum details", params(query("forum", "Forum shortname.")), "", "#/components/schemas/DisqusObject", "disqusAccessToken")},
			"/forums/listCategories.json": {"get": op("listDisqusForumCategories", "List forum categories", params(query("forum", "Forum shortname.")), "", "#/components/schemas/DisqusCollection", "disqusAccessToken")},
			"/forums/listPosts.json":      {"get": op("listDisqusForumPosts", "List forum posts", params(query("forum", "Forum shortname."), query("limit", "Maximum results."), query("cursor", "Pagination cursor.")), "", "#/components/schemas/DisqusCollection", "disqusAccessToken")},
			"/forums/listThreads.json":    {"get": op("listDisqusForumThreads", "List forum threads", params(query("forum", "Forum shortname."), query("limit", "Maximum results."), query("cursor", "Pagination cursor.")), "", "#/components/schemas/DisqusCollection", "disqusAccessToken")},
			"/threads/details.json":       {"get": op("getDisqusThread", "Get thread details", params(query("thread", "Thread identifier.")), "", "#/components/schemas/DisqusObject", "disqusAccessToken")},
			"/threads/listPosts.json":     {"get": op("listDisqusThreadPosts", "List thread posts", params(query("thread", "Thread identifier."), query("limit", "Maximum results."), query("cursor", "Pagination cursor.")), "", "#/components/schemas/DisqusCollection", "disqusAccessToken")},
		},
	}
}

func mediumOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mediumBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Medium access token", "description": "Medium OAuth or integration token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "medium",
		Title:       "Medium API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Medium human API documentation. This is not an official Medium OpenAPI document.",
		ServerURL:   "https://api.medium.com/v1",
		Sources:     []string{"https://github.com/Medium/medium-api-docs"},
		SourceNote:  "Medium publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected user, publication, and post endpoints.",
		Security:    security,
		Schemas:     []string{"MediumObject", "MediumCollection", "MediumError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/medium-api-overlay.json",
		Paths: map[string]map[string]any{
			"/me":                                  {"get": op("getMediumCurrentUser", "Get current user", nil, "", "#/components/schemas/MediumObject", "mediumBearer")},
			"/publications/{publication_id}/posts": {"post": op("createMediumPublicationPost", "Create publication post", params(path("publication_id", "Publication ID.")), "#/components/schemas/MediumObject", "#/components/schemas/MediumObject", "mediumBearer")},
			"/users/{user_id}/posts":               {"post": op("createMediumUserPost", "Create user post", params(path("user_id", "User ID.")), "#/components/schemas/MediumObject", "#/components/schemas/MediumObject", "mediumBearer")},
			"/users/{user_id}/publications":        {"get": op("listMediumUserPublications", "List user publications", params(path("user_id", "User ID.")), "", "#/components/schemas/MediumCollection", "mediumBearer")},
		},
	}
}

func npmOverlay() overlaySpec {
	security := map[string]map[string]any{
		"npmBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "npm access token", "description": "npm registry access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "npm",
		Title:       "npm Registry API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official npm registry human documentation. This is not an official npm OpenAPI document.",
		ServerURL:   "https://registry.npmjs.org",
		Sources:     []string{"https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md", "https://docs.npmjs.com/cli/v11/using-npm/registry"},
		SourceNote:  "npm publishes registry API documentation but no recorded stable public official OpenAPI document; this overlay covers selected package metadata, search, dist-tag, and whoami endpoints.",
		Security:    security,
		Schemas:     []string{"NpmObject", "NpmCollection", "NpmError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/npm-registry-api-overlay.json",
		Paths: map[string]map[string]any{
			"/-/package/{package_name}/dist-tags":                 {"get": op("listNpmDistTags", "List package dist-tags", params(path("package_name", "Package name.")), "", "#/components/schemas/NpmObject", "npmBearer")},
			"/-/package/{package_name}/dist-tags/{dist_tag_name}": {"put": op("setNpmDistTag", "Set package dist-tag", params(path("package_name", "Package name."), path("dist_tag_name", "Dist-tag name.")), "#/components/schemas/NpmObject", "#/components/schemas/NpmObject", "npmBearer")},
			"/-/v1/search":                      {"get": op("searchNpmPackages", "Search packages", params(query("text", "Search text."), query("size", "Maximum results."), query("from", "Pagination offset.")), "", "#/components/schemas/NpmCollection", "npmBearer")},
			"/-/whoami":                         {"get": op("getNpmWhoami", "Get authenticated npm user", nil, "", "#/components/schemas/NpmObject", "npmBearer")},
			"/{package_name}":                   {"get": op("getNpmPackageMetadata", "Get package metadata", params(path("package_name", "Package name.")), "", "#/components/schemas/NpmObject", "npmBearer")},
			"/{package_name}/{package_version}": {"get": op("getNpmPackageVersion", "Get package version metadata", params(path("package_name", "Package name."), path("package_version", "Package version.")), "", "#/components/schemas/NpmObject", "npmBearer")},
		},
	}
}

func storyblokOverlay() overlaySpec {
	security := map[string]map[string]any{
		"storyblokContentToken":     {"type": "apiKey", "in": "query", "name": "token", "description": "Storyblok Content Delivery API token carried in the token query parameter."},
		"storyblokManagementBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Storyblok personal access token", "description": "Storyblok Management API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "storyblok",
		Title:       "Storyblok API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Storyblok human API documentation. This is not an official Storyblok OpenAPI document.",
		ServerURL:   "https://api.storyblok.com",
		Sources:     []string{"https://www.storyblok.com/docs/api/content-delivery/v2", "https://www.storyblok.com/docs/api/management"},
		SourceNote:  "Storyblok publishes human Content Delivery and Management API documentation but no recorded stable public official OpenAPI document; this overlay covers selected story endpoints.",
		Security:    security,
		Schemas:     []string{"StoryblokObject", "StoryblokCollection", "StoryblokError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/storyblok-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v2/cdn/stories":                                    {"get": op("listStoryblokContentStories", "List content stories", params(query("starts_with", "Story path prefix."), query("version", "Content version.")), "", "#/components/schemas/StoryblokCollection", "storyblokContentToken")},
			"/v2/cdn/stories/{story_slug}":                       {"get": op("getStoryblokContentStory", "Get content story", params(path("story_slug", "Story slug."), query("version", "Content version.")), "", "#/components/schemas/StoryblokObject", "storyblokContentToken")},
			"/v1/spaces/{space_id}/stories":                      {"get": op("listStoryblokManagementStories", "List management stories", params(path("space_id", "Space ID.")), "", "#/components/schemas/StoryblokCollection", "storyblokManagementBearer"), "post": op("createStoryblokManagementStory", "Create management story", params(path("space_id", "Space ID.")), "#/components/schemas/StoryblokObject", "#/components/schemas/StoryblokObject", "storyblokManagementBearer")},
			"/v1/spaces/{space_id}/stories/{story_id}":           {"get": op("getStoryblokManagementStory", "Get management story", params(path("space_id", "Space ID."), path("story_id", "Story ID.")), "", "#/components/schemas/StoryblokObject", "storyblokManagementBearer"), "delete": op("deleteStoryblokManagementStory", "Delete management story", params(path("space_id", "Space ID."), path("story_id", "Story ID.")), "", "#/components/schemas/StoryblokObject", "storyblokManagementBearer")},
			"/v1/spaces/{space_id}/stories/{story_id}/publish":   {"get": op("publishStoryblokStory", "Publish story", params(path("space_id", "Space ID."), path("story_id", "Story ID.")), "", "#/components/schemas/StoryblokObject", "storyblokManagementBearer")},
			"/v1/spaces/{space_id}/stories/{story_id}/unpublish": {"get": op("unpublishStoryblokStory", "Unpublish story", params(path("space_id", "Space ID."), path("story_id", "Story ID.")), "", "#/components/schemas/StoryblokObject", "storyblokManagementBearer")},
		},
	}
}

func strapiOverlay() overlaySpec {
	security := map[string]map[string]any{
		"strapiBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Strapi JWT or API token", "description": "Strapi JWT or API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "strapi",
		Title:       "Strapi REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Strapi human API documentation. This is not an official Strapi OpenAPI document.",
		ServerURL:   "https://{strapi_host}/api",
		Sources:     []string{"https://docs.strapi.io/cms/api/rest", "https://docs.strapi.io/cms/plugins/documentation"},
		SourceNote:  "Strapi publishes REST API documentation and app-local OpenAPI generation guidance but no stable provider-wide public OpenAPI document; this overlay covers generic collection routes.",
		Security:    security,
		Schemas:     []string{"StrapiObject", "StrapiCollection", "StrapiError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/strapi-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/auth/local":                {"post": op("createStrapiLocalAuth", "Create local auth token", nil, "#/components/schemas/StrapiObject", "#/components/schemas/StrapiObject")},
			"/{content_type}":            {"get": op("listStrapiEntries", "List entries", params(path("content_type", "Collection type API ID."), query("filters", "Filter expression."), query("sort", "Sort expression."), query("pagination", "Pagination expression.")), "", "#/components/schemas/StrapiCollection", "strapiBearer"), "post": op("createStrapiEntry", "Create entry", params(path("content_type", "Collection type API ID.")), "#/components/schemas/StrapiObject", "#/components/schemas/StrapiObject", "strapiBearer")},
			"/{content_type}/{entry_id}": {"get": op("getStrapiEntry", "Get entry", params(path("content_type", "Collection type API ID."), path("entry_id", "Entry ID.")), "", "#/components/schemas/StrapiObject", "strapiBearer"), "put": op("updateStrapiEntry", "Update entry", params(path("content_type", "Collection type API ID."), path("entry_id", "Entry ID.")), "#/components/schemas/StrapiObject", "#/components/schemas/StrapiObject", "strapiBearer"), "delete": op("deleteStrapiEntry", "Delete entry", params(path("content_type", "Collection type API ID."), path("entry_id", "Entry ID.")), "", "#/components/schemas/StrapiObject", "strapiBearer")},
		},
	}
}

func wordpressOverlay() overlaySpec {
	security := map[string]map[string]any{
		"wordpressBasic":  {"type": "http", "scheme": "basic", "description": "WordPress username and application password or password carried with HTTP Basic authentication."},
		"wordpressBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "WordPress.com OAuth token", "description": "WordPress.com OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "wordpress",
		Title:       "WordPress REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official WordPress REST API documentation. This is not an official WordPress OpenAPI document.",
		ServerURL:   "https://{wordpress_host}/wp-json/wp/v2",
		Sources:     []string{"https://developer.wordpress.org/rest-api/", "https://developer.wordpress.org/rest-api/reference/"},
		SourceNote:  "WordPress publishes REST API human documentation but no stable provider-wide public OpenAPI document; this overlay covers selected core wp/v2 resources.",
		Security:    security,
		Schemas:     []string{"WordPressObject", "WordPressCollection", "WordPressError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/wordpress-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/categories":      {"get": op("listWordPressCategories", "List categories", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressCategory", "Create category", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/media":           {"get": op("listWordPressMedia", "List media", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressMedia", "Create media", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/pages":           {"get": op("listWordPressPages", "List pages", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressPage", "Create page", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/pages/{page_id}": {"get": op("getWordPressPage", "Get page", params(path("page_id", "Page ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic"), "post": op("updateWordPressPage", "Update page", params(path("page_id", "Page ID.")), "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic"), "delete": op("deleteWordPressPage", "Delete page", params(path("page_id", "Page ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/posts":           {"get": op("listWordPressPosts", "List posts", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressPost", "Create post", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/posts/{post_id}": {"get": op("getWordPressPost", "Get post", params(path("post_id", "Post ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic"), "post": op("updateWordPressPost", "Update post", params(path("post_id", "Post ID.")), "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic"), "delete": op("deleteWordPressPost", "Delete post", params(path("post_id", "Post ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/tags":            {"get": op("listWordPressTags", "List tags", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressTag", "Create tag", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/users":           {"get": op("listWordPressUsers", "List users", nil, "", "#/components/schemas/WordPressCollection", "wordpressBasic"), "post": op("createWordPressUser", "Create user", nil, "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic")},
			"/users/{user_id}": {"get": op("getWordPressUser", "Get user", params(path("user_id", "User ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic"), "post": op("updateWordPressUser", "Update user", params(path("user_id", "User ID.")), "#/components/schemas/WordPressObject", "#/components/schemas/WordPressObject", "wordpressBasic"), "delete": op("deleteWordPressUser", "Delete user", params(path("user_id", "User ID.")), "", "#/components/schemas/WordPressObject", "wordpressBasic")},
		},
	}
}

func wufooOverlay() overlaySpec {
	security := map[string]map[string]any{
		"wufooBasic": {"type": "http", "scheme": "basic", "description": "Wufoo API key supplied as HTTP Basic username with any placeholder password."},
	}
	return overlaySpec{
		ProviderID:  "wufoo",
		Title:       "Wufoo API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Wufoo human API documentation. This is not an official Wufoo OpenAPI document.",
		ServerURL:   "https://{subdomain}.wufoo.com/api/v3",
		Sources:     []string{"https://www.wufoo.com/docs/api/v3/", "https://www.wufoo.com/docs/api/v3/forms/"},
		SourceNote:  "Wufoo publishes API v3 human documentation but no recorded stable public official OpenAPI document; this overlay covers selected forms, entries, reports, users, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"WufooObject", "WufooCollection", "WufooError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/wufoo-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/forms.json":                                   {"get": op("listWufooForms", "List forms", nil, "", "#/components/schemas/WufooCollection", "wufooBasic")},
			"/forms/{form_hash}.json":                       {"get": op("getWufooForm", "Get form", params(path("form_hash", "Form hash.")), "", "#/components/schemas/WufooObject", "wufooBasic")},
			"/forms/{form_hash}/entries.json":               {"get": op("listWufooEntries", "List form entries", params(path("form_hash", "Form hash.")), "", "#/components/schemas/WufooCollection", "wufooBasic"), "post": op("createWufooEntry", "Create form entry", params(path("form_hash", "Form hash.")), "#/components/schemas/WufooObject", "#/components/schemas/WufooObject", "wufooBasic")},
			"/forms/{form_hash}/webhooks.json":              {"get": op("listWufooWebhooks", "List form webhooks", params(path("form_hash", "Form hash.")), "", "#/components/schemas/WufooCollection", "wufooBasic"), "put": op("createWufooWebhook", "Create form webhook", params(path("form_hash", "Form hash.")), "#/components/schemas/WufooObject", "#/components/schemas/WufooObject", "wufooBasic")},
			"/forms/{form_hash}/webhooks/{webhook_id}.json": {"delete": op("deleteWufooWebhook", "Delete form webhook", params(path("form_hash", "Form hash."), path("webhook_id", "Webhook ID.")), "", "#/components/schemas/WufooObject", "wufooBasic")},
			"/reports.json":                                 {"get": op("listWufooReports", "List reports", nil, "", "#/components/schemas/WufooCollection", "wufooBasic")},
			"/users.json":                                   {"get": op("listWufooUsers", "List users", nil, "", "#/components/schemas/WufooCollection", "wufooBasic")},
		},
	}
}

func yourlsOverlay() overlaySpec {
	security := map[string]map[string]any{
		"yourlsSignature": {"type": "apiKey", "in": "query", "name": "signature", "description": "YOURLS passwordless API signature carried in the signature query parameter."},
	}
	return overlaySpec{
		ProviderID:  "yourls",
		Title:       "YOURLS API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official YOURLS human API documentation. This is not an official YOURLS OpenAPI document.",
		ServerURL:   "https://{yourls_host}",
		Sources:     []string{"https://yourls.org/docs/guide/advanced/api", "https://yourls.org/docs/guide/advanced/passwordless-api"},
		SourceNote:  "YOURLS publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected shorturl, expand, stats, and URL stats actions.",
		Security:    security,
		Schemas:     []string{"YOURLSObject", "YOURLError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/yourls-api-overlay.json",
		Paths: map[string]map[string]any{
			"/yourls-api.php": {"get": op("callYOURLSAPI", "Call YOURLS API action", params(query("action", "YOURLS API action such as shorturl, expand, stats, or url-stats."), query("url", "URL for shorten or stats actions."), query("shorturl", "Short URL keyword for expand or stats actions."), query("format", "Response format.")), "", "#/components/schemas/YOURLSObject", "yourlsSignature")},
		},
	}
}

func raindropOverlay() overlaySpec {
	security := map[string]map[string]any{
		"raindropBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Raindrop.io OAuth token", "description": "Raindrop.io OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "raindrop",
		Title:       "Raindrop.io API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Raindrop.io human API documentation. This is not an official Raindrop.io OpenAPI document.",
		ServerURL:   "https://api.raindrop.io/rest/v1",
		Sources:     []string{"https://developer.raindrop.io/v1/", "https://developer.raindrop.io/v1/authentication"},
		SourceNote:  "Raindrop.io publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected raindrop, collection, tag, filter, and highlight endpoints.",
		Security:    security,
		Schemas:     []string{"RaindropObject", "RaindropCollection", "RaindropError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/raindrop-api-overlay.json",
		Paths: map[string]map[string]any{
			"/collection":                 {"get": op("listRaindropCollections", "List collections", nil, "", "#/components/schemas/RaindropCollection", "raindropBearer"), "post": op("createRaindropCollection", "Create collection", nil, "#/components/schemas/RaindropObject", "#/components/schemas/RaindropObject", "raindropBearer")},
			"/collection/{collection_id}": {"get": op("getRaindropCollection", "Get collection", params(path("collection_id", "Collection ID.")), "", "#/components/schemas/RaindropObject", "raindropBearer"), "put": op("updateRaindropCollection", "Update collection", params(path("collection_id", "Collection ID.")), "#/components/schemas/RaindropObject", "#/components/schemas/RaindropObject", "raindropBearer"), "delete": op("deleteRaindropCollection", "Delete collection", params(path("collection_id", "Collection ID.")), "", "#/components/schemas/RaindropObject", "raindropBearer")},
			"/filters/{collection_id}":    {"get": op("getRaindropFilters", "Get filters", params(path("collection_id", "Collection ID.")), "", "#/components/schemas/RaindropObject", "raindropBearer")},
			"/highlights/{raindrop_id}":   {"get": op("listRaindropHighlights", "List highlights", params(path("raindrop_id", "Raindrop ID.")), "", "#/components/schemas/RaindropCollection", "raindropBearer")},
			"/raindrop/{raindrop_id}":     {"get": op("getRaindrop", "Get raindrop", params(path("raindrop_id", "Raindrop ID.")), "", "#/components/schemas/RaindropObject", "raindropBearer"), "put": op("updateRaindrop", "Update raindrop", params(path("raindrop_id", "Raindrop ID.")), "#/components/schemas/RaindropObject", "#/components/schemas/RaindropObject", "raindropBearer"), "delete": op("deleteRaindrop", "Delete raindrop", params(path("raindrop_id", "Raindrop ID.")), "", "#/components/schemas/RaindropObject", "raindropBearer")},
			"/raindrops/{collection_id}":  {"get": op("listRaindrops", "List raindrops", params(path("collection_id", "Collection ID."), query("search", "Search query.")), "", "#/components/schemas/RaindropCollection", "raindropBearer"), "post": op("createRaindrop", "Create raindrop", params(path("collection_id", "Collection ID.")), "#/components/schemas/RaindropObject", "#/components/schemas/RaindropObject", "raindropBearer")},
			"/tags/{collection_id}":       {"get": op("listRaindropTags", "List tags", params(path("collection_id", "Collection ID.")), "", "#/components/schemas/RaindropCollection", "raindropBearer")},
			"/user":                       {"get": op("getRaindropUser", "Get user", nil, "", "#/components/schemas/RaindropObject", "raindropBearer")},
		},
	}
}

func travisCIOverlay() overlaySpec {
	security := map[string]map[string]any{
		"travisAuthorization": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "Travis CI API token carried in the Authorization header using token syntax."},
		"travisAPIVersion":    {"type": "apiKey", "in": "header", "name": "Travis-API-Version", "description": "Travis CI API version header required by API v3 requests."},
	}
	return overlaySpec{
		ProviderID:  "travis-ci",
		Title:       "Travis CI API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Travis CI human API documentation. This is not an official Travis CI OpenAPI document.",
		ServerURL:   "https://api.travis-ci.com",
		Sources:     []string{"https://developer.travis-ci.com/", "https://developer.travis-ci.com/authentication"},
		SourceNote:  "Travis CI publishes human API v3 documentation but no recorded stable public official OpenAPI document; this overlay covers selected repositories, builds, jobs, requests, branches, and environment-variable endpoints.",
		Security:    security,
		Schemas:     []string{"TravisCIObject", "TravisCICollection", "TravisCIError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/travis-ci-api-overlay.json",
		Paths: map[string]map[string]any{
			"/repo/{repository_id}":                      {"get": op("getTravisCIRepository", "Get repository", params(path("repository_id", "Repository ID or slug.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/repo/{repository_id}/branch/{branch_name}": {"get": op("getTravisCIBranch", "Get branch", params(path("repository_id", "Repository ID or slug."), path("branch_name", "Branch name.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/repo/{repository_id}/builds":               {"get": op("listTravisCIBuilds", "List builds", params(path("repository_id", "Repository ID or slug.")), "", "#/components/schemas/TravisCICollection", "travisAuthorization", "travisAPIVersion")},
			"/repo/{repository_id}/env_vars":             {"get": op("listTravisCIEnvironmentVariables", "List environment variables", params(path("repository_id", "Repository ID or slug.")), "", "#/components/schemas/TravisCICollection", "travisAuthorization", "travisAPIVersion"), "post": op("createTravisCIEnvironmentVariable", "Create environment variable", params(path("repository_id", "Repository ID or slug.")), "#/components/schemas/TravisCIObject", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/repo/{repository_id}/env_var/{env_var_id}": {"get": op("getTravisCIEnvironmentVariable", "Get environment variable", params(path("repository_id", "Repository ID or slug."), path("env_var_id", "Environment variable ID.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion"), "delete": op("deleteTravisCIEnvironmentVariable", "Delete environment variable", params(path("repository_id", "Repository ID or slug."), path("env_var_id", "Environment variable ID.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/repo/{repository_id}/request":              {"post": op("createTravisCIRequest", "Create build request", params(path("repository_id", "Repository ID or slug.")), "#/components/schemas/TravisCIObject", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/build/{build_id}":                          {"get": op("getTravisCIBuild", "Get build", params(path("build_id", "Build ID.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/job/{job_id}":                              {"get": op("getTravisCIJob", "Get job", params(path("job_id", "Job ID.")), "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
			"/repos":                                     {"get": op("listTravisCIRepositories", "List repositories", nil, "", "#/components/schemas/TravisCICollection", "travisAuthorization", "travisAPIVersion")},
			"/user":                                      {"get": op("getTravisCIUser", "Get user", nil, "", "#/components/schemas/TravisCIObject", "travisAuthorization", "travisAPIVersion")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-20",
			"description": spec.Description,
		},
		"servers": []map[string]any{{"url": spec.ServerURL}},
		"paths":   orderedMap(spec.Paths),
		"components": map[string]any{
			"securitySchemes": orderedMap(spec.Security),
			"schemas":         schemas(spec.Schemas),
		},
		"x-apitools-overlay": map[string]any{
			"provider_id":       spec.ProviderID,
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs":       spec.Sources,
			"source_note":       spec.SourceNote,
		},
	}
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, securityNames ...string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200":     response(responseRef),
			"default": map[string]any{"description": "Provider error response."},
		},
	}
	if len(securityNames) > 0 {
		requirement := map[string][]string{}
		for _, name := range securityNames {
			if name != "" {
				requirement[name] = []string{}
			}
		}
		if len(requirement) > 0 {
			out["security"] = []map[string][]string{requirement}
		}
	}
	if len(parameters) > 0 {
		out["parameters"] = parameters
	}
	if requestRef != "" {
		out["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": requestRef}}},
		}
	}
	return out
}

func response(ref string) map[string]any {
	out := map[string]any{"description": "Successful response."}
	if ref != "" {
		out["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": ref}}}
	}
	return out
}

func params(items ...map[string]any) []map[string]any { return items }

func path(name, description string) map[string]any { return parameter(name, "path", description, true) }

func query(name, description string) map[string]any {
	return parameter(name, "query", description, false)
}

func parameter(name, in, description string, required bool) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": "string"},
	}
}

func schemas(names []string) map[string]map[string]any {
	sort.Strings(names)
	out := map[string]map[string]any{}
	for _, name := range names {
		out[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return out
}

func orderedMap[V any](in map[string]V) map[string]V {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := map[string]V{}
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func write(path string, doc map[string]any) {
	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		panic(err)
	}
}
