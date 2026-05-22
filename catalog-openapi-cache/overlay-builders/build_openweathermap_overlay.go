//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "OpenWeatherMap One Call 3.0 and Geocoding Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official OpenWeatherMap One Call 3.0 and Geocoding API human documentation. This is not an official OpenWeatherMap OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.openweathermap.org"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "openweathermap",
			"overlay_id":        "openweathermap-one-call-3-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://openweathermap.org/api/one-call-3?collection=one_call_api_3.0",
				"https://openweathermap.org/api/geocoding-api",
				"https://openweathermap.org/appid",
			},
			"source_note": "OpenWeatherMap publishes human documentation for One Call API 3.0, Geocoding API coordinate lookup helpers, and API key usage, but no official OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"openWeatherAPIKey": map[string]any{
					"type":        "apiKey",
					"in":          "query",
					"name":        "appid",
					"description": "OpenWeather API key passed as the appid query parameter.",
				},
			},
			"schemas": map[string]any{
				"OneCallResponse": map[string]any{
					"type":        "object",
					"description": "Partial advisory schema for documented One Call API 3.0 top-level response fields.",
					"properties": map[string]any{
						"lat":             map[string]any{"type": "number"},
						"lon":             map[string]any{"type": "number"},
						"timezone":        map[string]any{"type": "string"},
						"timezone_offset": map[string]any{"type": "integer"},
						"current":         map[string]any{"type": "object", "additionalProperties": true},
						"minutely":        map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
						"hourly":          map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
						"daily":           map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
						"alerts":          map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
					},
				},
				"GeocodingLocation": map[string]any{
					"type":        "object",
					"description": "Advisory schema for a documented OpenWeatherMap geocoding location result. Fields vary by country and location.",
					"properties": map[string]any{
						"name":        map[string]any{"type": "string", "description": "Name of the found location."},
						"local_names": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "Location names in available languages; keys are language codes or provider-specific internal keys."},
						"lat":         map[string]any{"type": "number", "description": "Geographical coordinates of the found location: latitude."},
						"lon":         map[string]any{"type": "number", "description": "Geographical coordinates of the found location: longitude."},
						"country":     map[string]any{"type": "string", "description": "Country code of the found location."},
						"state":       map[string]any{"type": "string", "description": "State of the found location, where available."},
					},
				},
				"ZipGeocodingResponse": map[string]any{
					"type":        "object",
					"description": "Advisory schema for a documented OpenWeatherMap zip/post-code geocoding result.",
					"properties": map[string]any{
						"zip":     map[string]any{"type": "string", "description": "Zip/post code specified in the API request."},
						"name":    map[string]any{"type": "string", "description": "Name of the found area."},
						"lat":     map[string]any{"type": "number", "description": "Geographical coordinates of the centroid of the found zip/post code: latitude."},
						"lon":     map[string]any{"type": "number", "description": "Geographical coordinates of the centroid of the found zip/post code: longitude."},
						"country": map[string]any{"type": "string", "description": "Country code of the found zip/post code."},
					},
				},
				"OpenWeatherError": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
		},
		"security": []map[string]any{{"openWeatherAPIKey": []string{}}},
		"paths": map[string]any{
			"/data/3.0/onecall": map[string]any{
				"get": map[string]any{
					"operationId": "getOpenWeatherMapOneCall3",
					"summary":     "Get One Call API 3.0 weather data",
					"description": "Advisory operation derived from official OpenWeatherMap One Call API 3.0 documentation.",
					"parameters": []map[string]any{
						{"name": "lat", "in": "query", "required": true, "schema": map[string]any{"type": "number", "minimum": -90, "maximum": 90}, "description": "Latitude in decimal degrees, from -90 to 90. OpenWeatherMap directs callers that need to convert city names or zip codes to coordinates, or coordinates back to location names, to the Geocoding API."},
						{"name": "lon", "in": "query", "required": true, "schema": map[string]any{"type": "number", "minimum": -180, "maximum": 180}, "description": "Longitude in decimal degrees, from -180 to 180. OpenWeatherMap directs callers that need to convert city names or zip codes to coordinates, or coordinates back to location names, to the Geocoding API."},
						{"name": "appid", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "OpenWeather API key."},
						{"name": "exclude", "in": "query", "required": false, "schema": map[string]any{"type": "string"}, "description": "Comma-separated weather data blocks to exclude, such as current,minutely,hourly,daily,alerts."},
						{"name": "units", "in": "query", "required": false, "schema": map[string]any{"type": "string", "enum": []string{"standard", "metric", "imperial"}}, "description": "Units of measurement."},
						{"name": "lang", "in": "query", "required": false, "schema": map[string]any{"type": "string"}, "description": "Language code."},
					},
					"security": []map[string]any{{"openWeatherAPIKey": []string{}}},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "One Call API 3.0 response.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/OneCallResponse"}}},
						},
						"default": map[string]any{
							"description": "OpenWeather error response.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/OpenWeatherError"}}},
						},
					},
				},
			},
			"/geo/1.0/direct": map[string]any{
				"get": map[string]any{
					"operationId": "geocodeOpenWeatherMapLocationName",
					"summary":     "Geocode location name",
					"description": "Advisory operation derived from official OpenWeatherMap Geocoding API documentation. Direct geocoding converts a city or area name to geographical coordinates.",
					"parameters": []map[string]any{
						{"name": "q", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "City name, optional US state code, and optional country code separated by commas. Use ISO 3166 country codes."},
						{"name": "appid", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "OpenWeather API key."},
						{"name": "limit", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 5}, "description": "Maximum number of matching locations to return, up to 5."},
					},
					"security": []map[string]any{{"openWeatherAPIKey": []string{}}},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "List of matching geocoding locations.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/GeocodingLocation"}}}},
						},
						"default": map[string]any{
							"description": "OpenWeather error response.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/OpenWeatherError"}}},
						},
					},
				},
			},
			"/geo/1.0/zip": map[string]any{
				"get": map[string]any{
					"operationId": "geocodeOpenWeatherMapZipCode",
					"summary":     "Geocode zip or post code",
					"description": "Advisory operation derived from official OpenWeatherMap Geocoding API documentation. Zip geocoding converts a zip/post code and country code to geographical coordinates.",
					"parameters": []map[string]any{
						{"name": "zip", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "Zip/post code and country code separated by a comma. Use ISO 3166 country codes."},
						{"name": "appid", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "OpenWeather API key."},
					},
					"security": []map[string]any{{"openWeatherAPIKey": []string{}}},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Zip or post-code geocoding result.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ZipGeocodingResponse"}}},
						},
						"default": map[string]any{
							"description": "OpenWeather error response.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/OpenWeatherError"}}},
						},
					},
				},
			},
			"/geo/1.0/reverse": map[string]any{
				"get": map[string]any{
					"operationId": "reverseGeocodeOpenWeatherMapCoordinates",
					"summary":     "Reverse geocode coordinates",
					"description": "Advisory operation derived from official OpenWeatherMap Geocoding API documentation. Reverse geocoding converts geographical coordinates to nearby location names.",
					"parameters": []map[string]any{
						{"name": "lat", "in": "query", "required": true, "schema": map[string]any{"type": "number"}, "description": "Geographical coordinates: latitude."},
						{"name": "lon", "in": "query", "required": true, "schema": map[string]any{"type": "number"}, "description": "Geographical coordinates: longitude."},
						{"name": "appid", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "OpenWeather API key."},
						{"name": "limit", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1}, "description": "Maximum number of location names to return."},
					},
					"security": []map[string]any{{"openWeatherAPIKey": []string{}}},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "List of nearby geocoding locations.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/GeocodingLocation"}}}},
						},
						"default": map[string]any{
							"description": "OpenWeather error response.",
							"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/OpenWeatherError"}}},
						},
					},
				},
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/openweathermap-one-call-3-overlay.json", doc)
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
