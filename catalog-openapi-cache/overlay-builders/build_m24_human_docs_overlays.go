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
		coinGeckoOverlay(),
		hackerNewsOverlay(),
		nasaOverlay(),
		oneSimpleAPIOverlay(),
		openThesaurusOverlay(),
		quickChartOverlay(),
		redditOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func coinGeckoOverlay() overlaySpec {
	security := map[string]map[string]any{
		"coingeckoDemoAPIKeyHeader": {"type": "apiKey", "in": "header", "name": "x-cg-demo-api-key", "description": "CoinGecko Public/Demo API key carried in the x-cg-demo-api-key header."},
	}
	return overlaySpec{
		ProviderID:  "coingecko",
		Title:       "CoinGecko API v3 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official CoinGecko API human documentation. This is not an official CoinGecko OpenAPI document.",
		ServerURL:   "https://api.coingecko.com/api/v3",
		Sources:     []string{"https://docs.coingecko.com/reference/introduction", "https://docs.coingecko.com/reference/authentication", "https://docs.coingecko.com/v3.0.1/reference/authentication", "https://docs.coingecko.com/reference/endpoint-overview", "https://docs.coingecko.com/reference/simple-price"},
		SourceNote:  "CoinGecko publishes API reference docs with Public/Demo and Pro API-key metadata but no recorded official OpenAPI document; this overlay covers selected public v3 market-data endpoints using the Demo header form.",
		Security:    security,
		Schemas:     []string{"CoinGeckoObject", "CoinGeckoCollection", "CoinGeckoError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/coingecko-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/ping":          {"get": op("pingCoinGecko", "Check API server status", nil, "", "#/components/schemas/CoinGeckoObject", "coingeckoDemoAPIKeyHeader")},
			"/simple/price":  {"get": op("getCoinGeckoSimplePrice", "Get coin price by IDs, symbols, or names", params(query("ids", "Comma-separated CoinGecko coin IDs."), query("vs_currencies", "Comma-separated target currencies."), query("include_market_cap", "Whether to include market capitalization.")), "", "#/components/schemas/CoinGeckoObject", "coingeckoDemoAPIKeyHeader")},
			"/coins/list":    {"get": op("listCoinGeckoCoins", "List coin ID map", params(query("include_platform", "Whether to include platform contract addresses.")), "", "#/components/schemas/CoinGeckoCollection", "coingeckoDemoAPIKeyHeader")},
			"/coins/markets": {"get": op("listCoinGeckoCoinMarkets", "List coin markets", params(query("vs_currency", "Target currency."), query("ids", "Comma-separated coin IDs."), query("category", "Coin category filter."), query("per_page", "Page size."), query("page", "Page number.")), "", "#/components/schemas/CoinGeckoCollection", "coingeckoDemoAPIKeyHeader")},
			"/coins/{id}":    {"get": op("getCoinGeckoCoin", "Get coin data by ID", params(path("id", "CoinGecko coin ID."), query("localization", "Whether to include localized names."), query("tickers", "Whether to include tickers."), query("market_data", "Whether to include market data.")), "", "#/components/schemas/CoinGeckoObject", "coingeckoDemoAPIKeyHeader")},
			"/exchanges":     {"get": op("listCoinGeckoExchanges", "List exchanges", params(query("per_page", "Page size."), query("page", "Page number.")), "", "#/components/schemas/CoinGeckoCollection", "coingeckoDemoAPIKeyHeader")},
		},
	}
}

func hackerNewsOverlay() overlaySpec {
	return overlaySpec{
		ProviderID:  "hackernews",
		Title:       "Hacker News Firebase API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Hacker News API human documentation. This is not an official Hacker News OpenAPI document.",
		ServerURL:   "https://hacker-news.firebaseio.com/v0",
		Sources:     []string{"https://github.com/HackerNews/API"},
		SourceNote:  "Hacker News publishes public Firebase API docs for items, users, story lists, max item, and updates without authentication.",
		Schemas:     []string{"HackerNewsObject", "HackerNewsCollection", "HackerNewsError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/hackernews-firebase-api-overlay.json",
		Paths: map[string]map[string]any{
			"/item/{item_id}.json": {"get": op("getHackerNewsItem", "Get item", params(path("item_id", "Hacker News item ID.")), "", "#/components/schemas/HackerNewsObject")},
			"/user/{username}.json": {
				"get": op("getHackerNewsUser", "Get user", params(path("username", "Hacker News username.")), "", "#/components/schemas/HackerNewsObject"),
			},
			"/topstories.json":  {"get": op("listHackerNewsTopStories", "List top story IDs", nil, "", "#/components/schemas/HackerNewsCollection")},
			"/newstories.json":  {"get": op("listHackerNewsNewStories", "List new story IDs", nil, "", "#/components/schemas/HackerNewsCollection")},
			"/beststories.json": {"get": op("listHackerNewsBestStories", "List best story IDs", nil, "", "#/components/schemas/HackerNewsCollection")},
			"/updates.json":     {"get": op("getHackerNewsUpdates", "Get changed items and profiles", nil, "", "#/components/schemas/HackerNewsObject")},
		},
	}
}

func nasaOverlay() overlaySpec {
	security := map[string]map[string]any{
		"nasaAPIKey": {"type": "apiKey", "in": "query", "name": "api_key", "description": "NASA Open APIs key carried in the api_key query parameter."},
	}
	return overlaySpec{
		ProviderID:  "nasa",
		Title:       "NASA Open APIs Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official NASA Open APIs human documentation. This is not an official NASA OpenAPI document.",
		ServerURL:   "https://api.nasa.gov",
		Sources:     []string{"https://api.nasa.gov/"},
		SourceNote:  "NASA publishes Open APIs human docs and api_key signup metadata but no recorded official OpenAPI document; this overlay covers APOD, Mars Rover Photos, NeoWs, EPIC, and DONKI examples.",
		Security:    security,
		Schemas:     []string{"NASAObject", "NASACollection", "NASAError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/nasa-open-apis-overlay.json",
		Paths: map[string]map[string]any{
			"/planetary/apod": {"get": op("getNASAAstronomyPictureOfTheDay", "Get astronomy picture of the day", params(query("date", "APOD date in YYYY-MM-DD format."), query("start_date", "Start date for a date range."), query("end_date", "End date for a date range."), query("thumbs", "Whether to return video thumbnails.")), "", "#/components/schemas/NASAObject", "nasaAPIKey")},
			"/mars-photos/api/v1/rovers/{rover}/photos": {"get": op("listNASAMarsRoverPhotos", "List Mars rover photos", params(path("rover", "Mars rover name."), query("sol", "Martian sol."), query("earth_date", "Earth date in YYYY-MM-DD format."), query("camera", "Camera filter."), query("page", "Page number.")), "", "#/components/schemas/NASACollection", "nasaAPIKey")},
			"/neo/rest/v1/feed":                         {"get": op("getNASANearEarthObjectFeed", "Get near-earth object feed", params(query("start_date", "Feed start date."), query("end_date", "Feed end date.")), "", "#/components/schemas/NASAObject", "nasaAPIKey")},
			"/EPIC/api/natural":                         {"get": op("listNASAEarthPolychromaticImages", "List EPIC natural images", params(query("date", "Optional EPIC date.")), "", "#/components/schemas/NASACollection", "nasaAPIKey")},
			"/DONKI/notifications":                      {"get": op("listNASADONKINotifications", "List DONKI notifications", params(query("startDate", "Start date."), query("endDate", "End date."), query("type", "Notification type.")), "", "#/components/schemas/NASACollection", "nasaAPIKey")},
		},
	}
}

func oneSimpleAPIOverlay() overlaySpec {
	security := map[string]map[string]any{
		"oneSimpleAPIToken": {"type": "apiKey", "in": "query", "name": "token", "description": "OneSimpleApi access token carried in the token query parameter."},
	}
	return overlaySpec{
		ProviderID:  "onesimpleapi",
		Title:       "OneSimpleApi Toolkit Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official OneSimpleApi human documentation. This is not an official OneSimpleApi OpenAPI document.",
		ServerURL:   "https://onesimpleapi.com",
		Sources:     []string{"https://onesimpleapi.com/docs", "https://onesimpleapi.com/user/api-tokens"},
		SourceNote:  "OneSimpleApi publishes human docs with token query examples but no recorded official OpenAPI document; this overlay covers selected utility endpoints visible in the official docs.",
		Security:    security,
		Schemas:     []string{"OneSimpleAPIObject", "OneSimpleAPIError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/onesimpleapi-toolkit-overlay.json",
		Paths: map[string]map[string]any{
			"/api/page_info":   {"get": op("getOneSimpleAPIPageInfo", "Get web page metadata", params(query("url", "URL to analyze."), query("output", "Response output format."), query("headers", "Whether to include response headers.")), "", "#/components/schemas/OneSimpleAPIObject", "oneSimpleAPIToken")},
			"/api/page_status": {"get": op("getOneSimpleAPIPageStatus", "Get web page status and certificate info", params(query("url", "URL to analyze."), query("output", "Response output format.")), "", "#/components/schemas/OneSimpleAPIObject", "oneSimpleAPIToken")},
			"/api/screenshot":  {"get": op("createOneSimpleAPIScreenshot", "Create a page screenshot", params(query("url", "URL to capture."), query("output", "Response output format."), query("screen", "Viewport size.")), "", "#/components/schemas/OneSimpleAPIObject", "oneSimpleAPIToken")},
			"/api/pdf":         {"get": op("createOneSimpleAPIPDF", "Create a PDF from a URL", params(query("url", "URL to render."), query("output", "Response output format.")), "", "#/components/schemas/OneSimpleAPIObject", "oneSimpleAPIToken")},
			"/api/qr-code":     {"get": op("createOneSimpleAPIQRCode", "Create a QR code", params(query("data", "QR code payload."), query("output", "Response output format.")), "", "#/components/schemas/OneSimpleAPIObject", "oneSimpleAPIToken")},
		},
	}
}

func openThesaurusOverlay() overlaySpec {
	return overlaySpec{
		ProviderID:  "openthesaurus",
		Title:       "OpenThesaurus API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official OpenThesaurus API human documentation. This is not an official OpenThesaurus OpenAPI document.",
		ServerURL:   "https://www.openthesaurus.de",
		Sources:     []string{"https://www.openthesaurus.de/about/api"},
		SourceNote:  "OpenThesaurus publishes human webservice docs for anonymous synonym search with JSON and XML response formats.",
		Schemas:     []string{"OpenThesaurusObject", "OpenThesaurusError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/openthesaurus-api-overlay.json",
		Paths: map[string]map[string]any{
			"/synonyme/search": {"get": op("searchOpenThesaurusSynonyms", "Search synonyms", params(query("q", "Search term."), query("format", "Response media type such as application/json or text/xml."), query("similar", "Whether to return similarly spelled words."), query("substring", "Whether to return substring matches.")), "", "#/components/schemas/OpenThesaurusObject")},
		},
	}
}

func quickChartOverlay() overlaySpec {
	return overlaySpec{
		ProviderID:  "quickchart",
		Title:       "QuickChart API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official QuickChart human documentation. This is not an official QuickChart OpenAPI document.",
		ServerURL:   "https://quickchart.io",
		Sources:     []string{"https://quickchart.io/documentation/", "https://quickchart.io/documentation/usage/post-endpoint/", "https://quickchart.io/documentation/qr-codes/"},
		SourceNote:  "QuickChart publishes human API docs for anonymous chart rendering and QR-code endpoints but no recorded official OpenAPI document.",
		Schemas:     []string{"QuickChartRequest", "QuickChartObject", "QuickChartError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/quickchart-api-overlay.json",
		Paths: map[string]map[string]any{
			"/chart":    {"get": op("getQuickChartChart", "Render a chart by URL", params(query("c", "Chart.js configuration."), query("chart", "Chart.js configuration."), query("width", "Image width in pixels."), query("height", "Image height in pixels."), query("format", "Output format.")), "", "", ""), "post": op("createQuickChartChart", "Render a chart by POST", nil, "#/components/schemas/QuickChartRequest", "", "")},
			"/qr":       {"get": op("getQuickChartQRCode", "Render a QR code", params(query("text", "Text or URL to encode."), query("size", "QR image size."), query("format", "Output format.")), "", "", "")},
			"/graphviz": {"get": op("getQuickChartGraphviz", "Render a Graphviz diagram", params(query("graph", "Graphviz DOT source."), query("format", "Output format.")), "", "", "")},
		},
	}
}

func redditOverlay() overlaySpec {
	security := map[string]map[string]any{
		"redditBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "Reddit OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "reddit",
		Title:       "Reddit API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Reddit API human documentation. This is not an official Reddit OpenAPI document.",
		ServerURL:   "https://oauth.reddit.com",
		Sources:     []string{"https://www.reddit.com/dev/api/", "https://www.reddit.com/dev/api/oauth", "https://developers.reddit.com/docs/capabilities/server/reddit-api"},
		SourceNote:  "Reddit publishes generated API docs and Developer Platform server API docs but no recorded official OpenAPI document; this overlay covers selected OAuth API endpoints.",
		Security:    security,
		Schemas:     []string{"RedditObject", "RedditListing", "RedditError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/reddit-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/v1/me":             {"get": op("getRedditAuthenticatedUser", "Get authenticated user", nil, "", "#/components/schemas/RedditObject", "redditBearer")},
			"/user/{username}/about": {"get": op("getRedditUserAbout", "Get user about data", params(path("username", "Reddit username.")), "", "#/components/schemas/RedditObject", "redditBearer")},
			"/r/{subreddit}/about":   {"get": op("getRedditSubredditAbout", "Get subreddit about data", params(path("subreddit", "Subreddit name.")), "", "#/components/schemas/RedditObject", "redditBearer")},
			"/r/{subreddit}/new":     {"get": op("listRedditSubredditNew", "List new subreddit posts", params(path("subreddit", "Subreddit name."), query("after", "Pagination fullname after anchor."), query("before", "Pagination fullname before anchor."), query("limit", "Maximum items to return.")), "", "#/components/schemas/RedditListing", "redditBearer")},
			"/r/{subreddit}/hot":     {"get": op("listRedditSubredditHot", "List hot subreddit posts", params(path("subreddit", "Subreddit name."), query("after", "Pagination fullname after anchor."), query("before", "Pagination fullname before anchor."), query("limit", "Maximum items to return.")), "", "#/components/schemas/RedditListing", "redditBearer")},
			"/api/submit":            {"post": op("submitRedditPost", "Submit a post", params(query("api_type", "Response API type."), query("sr", "Target subreddit."), query("kind", "Post kind."), query("title", "Post title."), query("url", "Link URL."), query("text", "Self-post text.")), "", "#/components/schemas/RedditObject", "redditBearer")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-19",
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
	return doc
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, securityNames ...string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200": response(responseRef),
			"default": map[string]any{
				"description": "Provider error response.",
			},
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

func params(items ...map[string]any) []map[string]any {
	return items
}

func path(name, description string) map[string]any {
	return parameter(name, "path", description, true)
}

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
