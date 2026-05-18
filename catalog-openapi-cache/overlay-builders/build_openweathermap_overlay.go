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
			"title":       "OpenWeatherMap One Call 3.0 Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official OpenWeatherMap human documentation. This is not an official OpenWeatherMap OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.openweathermap.org"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "openweathermap",
			"overlay_id":        "openweathermap-one-call-3-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://openweathermap.org/api/one-call-3?collection=one_call_api_3.0",
				"https://openweathermap.org/appid",
			},
			"source_note": "OpenWeatherMap publishes human documentation for One Call API 3.0 and API key usage, but no official OpenAPI document is recorded in the apitools catalog.",
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
						{"name": "lat", "in": "query", "required": true, "schema": map[string]any{"type": "number", "minimum": -90, "maximum": 90}, "description": "Latitude."},
						{"name": "lon", "in": "query", "required": true, "schema": map[string]any{"type": "number", "minimum": -180, "maximum": 180}, "description": "Longitude."},
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
