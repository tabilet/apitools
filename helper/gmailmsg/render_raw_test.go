package gmailmsg

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/helper/fnctspec"
)

func TestRenderRawEncodesGmailRawMessage(t *testing.T) {
	raw, err := RenderRaw(RenderRawInput{
		To:      "user@example.com",
		From:    "sender@example.com",
		Subject: "Weather report",
		Body:    "It is 12 C in Toronto.",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	msg := string(decoded)
	for _, want := range []string{
		"From: sender@example.com\r\n",
		"To: user@example.com\r\n",
		"Subject: Weather report\r\n",
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n",
		"\r\nIt is 12 C in Toronto.",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("decoded message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(raw, "=") {
		t.Fatalf("raw message is padded: %q", raw)
	}
}

func TestRenderRawAnyRendersTemplateWithInput(t *testing.T) {
	raw, err := RenderRawAny(map[string]any{
		"to":            "user@example.com",
		"subject":       "Weather report",
		"body_template": "Current temperature: {{.current.temp}} C",
		"input": map[string]any{
			"current": map[string]any{"temp": 12},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decoded), "Current temperature: 12 C") {
		t.Fatalf("decoded message = %s", string(decoded))
	}
}

func TestRenderRawRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		in   RenderRawInput
		want string
	}{
		{name: "missing to", in: RenderRawInput{Subject: "s", Body: "b"}, want: "requires to"},
		{name: "missing subject", in: RenderRawInput{To: "a@example.com", Body: "b"}, want: "requires subject"},
		{name: "missing body", in: RenderRawInput{To: "a@example.com", Subject: "s"}, want: "requires body or body_template"},
		{name: "both body forms", in: RenderRawInput{To: "a@example.com", Subject: "s", Body: "b", BodyTemplate: "t"}, want: "either body or body_template"},
		{name: "header injection", in: RenderRawInput{To: "a@example.com\r\nBcc: b@example.com", Subject: "s", Body: "b"}, want: "must not contain CR or LF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderRaw(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RenderRaw() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRenderRawTemplateMissingKeyFails(t *testing.T) {
	_, err := RenderRaw(RenderRawInput{
		To:           "user@example.com",
		Subject:      "Weather",
		BodyTemplate: "{{.current.temp}} {{.missing.value}}",
		Input:        map[string]any{"current": map[string]any{"temp": 12}},
	})
	if err == nil || !strings.Contains(err.Error(), "render gmail body_template") {
		t.Fatalf("RenderRaw() error = %v", err)
	}
}

func TestRenderRawEncodesNonASCIIHeaders(t *testing.T) {
	raw, err := RenderRaw(RenderRawInput{
		To:      "Jose Alvarez <jose@example.com>",
		From:    "Jose Alvarez <sender@example.com>",
		Subject: "Weather cafe",
		Body:    "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	msg := string(decoded)
	if !strings.Contains(msg, "Subject: Weather cafe\r\n") {
		t.Fatalf("decoded message = %s", msg)
	}

	raw, err = RenderRaw(RenderRawInput{
		To:      "Jose Alvarez <jose@example.com>",
		Subject: "Weather café",
		Body:    "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decoded), "Subject: =?utf-8?q?Weather_caf=C3=A9?=\r\n") {
		t.Fatalf("decoded message = %s", string(decoded))
	}
}

func TestFunctionSpec(t *testing.T) {
	spec := FunctionSpec()
	if spec.Name != FunctionNameRenderRaw || spec.RuntimeType != "fnct" {
		t.Fatalf("spec identity = %#v", spec)
	}
	if spec.InvocationMode != fnctspec.InvocationRequestBodyObject {
		t.Fatalf("spec invocation = %q", spec.InvocationMode)
	}
	if !spec.Pure || spec.SideEffects != "none" {
		t.Fatalf("spec side effects = pure %v, %q", spec.Pure, spec.SideEffects)
	}
	if spec.Output.Name != "received_body" || spec.Output.Semantic != "gmail_raw_message" {
		t.Fatalf("spec output = %#v", spec.Output)
	}
}
