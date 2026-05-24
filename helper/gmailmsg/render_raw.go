package gmailmsg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"text/template"

	"github.com/OpenUdon/apitools/helper/fnctspec"
)

const FunctionNameRenderRaw = "gmail.render_raw"

// RenderRawInput is the request-body contract for gmail.render_raw.
type RenderRawInput struct {
	To           string `json:"to"`
	From         string `json:"from,omitempty"`
	Subject      string `json:"subject"`
	Body         string `json:"body,omitempty"`
	BodyTemplate string `json:"body_template,omitempty"`
	Input        any    `json:"input,omitempty"`
}

// FunctionSpec returns the public metadata for gmail.render_raw.
func FunctionSpec() fnctspec.FunctionSpec {
	return fnctspec.FunctionSpec{
		Name:           FunctionNameRenderRaw,
		RuntimeType:    "fnct",
		InvocationMode: fnctspec.InvocationRequestBodyObject,
		Summary:        "Render a Gmail API raw message string from recipient, subject, body/template, and optional input data.",
		Inputs: []fnctspec.FieldSpec{
			{Name: "to", Type: "string", Required: true, Semantic: "email_recipient", Description: "Recipient email address."},
			{Name: "from", Type: "string", Semantic: "email_sender", Description: "Optional sender header."},
			{Name: "subject", Type: "string", Required: true, Semantic: "email_subject", Description: "Email subject header."},
			{Name: "body", Type: "string", Semantic: "email_body", Description: "Literal plain-text body. Mutually exclusive with body_template."},
			{Name: "body_template", Type: "string", Semantic: "email_body_template", Description: "Go text/template body rendered with input as dot. Mutually exclusive with body."},
			{Name: "input", Type: "any", Semantic: "template_input", Description: "Optional data used as the template dot."},
		},
		Output:      fnctspec.FieldSpec{Name: "received_body", Type: "string", Semantic: "gmail_raw_message", Description: "Base64url encoded Gmail raw message."},
		Pure:        true,
		SideEffects: "none",
	}
}

// RenderRawAny coerces a request-body object into RenderRawInput and renders a
// Gmail API raw message string.
func RenderRawAny(value any) (string, error) {
	var input RenderRawInput
	switch typed := value.(type) {
	case RenderRawInput:
		input = typed
	case *RenderRawInput:
		if typed == nil {
			return "", errors.New("gmail render raw input is nil")
		}
		input = *typed
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("marshal gmail render raw input: %w", err)
		}
		if err := json.Unmarshal(data, &input); err != nil {
			return "", fmt.Errorf("decode gmail render raw input: %w", err)
		}
	}
	return RenderRaw(input)
}

// RenderRaw renders a Gmail API raw message string. The returned value is the
// unpadded base64url encoding expected by users.messages.send.raw.
func RenderRaw(input RenderRawInput) (string, error) {
	if strings.TrimSpace(input.To) == "" {
		return "", errors.New("gmail render raw requires to")
	}
	if strings.TrimSpace(input.Subject) == "" {
		return "", errors.New("gmail render raw requires subject")
	}
	to, err := formatAddressHeader("to", input.To)
	if err != nil {
		return "", err
	}
	from, err := formatAddressHeader("from", input.From)
	if err != nil {
		return "", err
	}
	if err := validateHeader("subject", input.Subject); err != nil {
		return "", err
	}
	hasBody := input.Body != ""
	hasTemplate := input.BodyTemplate != ""
	switch {
	case hasBody && hasTemplate:
		return "", errors.New("gmail render raw accepts either body or body_template, not both")
	case !hasBody && !hasTemplate:
		return "", errors.New("gmail render raw requires body or body_template")
	}

	body := input.Body
	if hasTemplate {
		rendered, err := renderTemplate(input.BodyTemplate, input.Input)
		if err != nil {
			return "", err
		}
		body = rendered
	}

	var msg strings.Builder
	if from != "" {
		fmt.Fprintf(&msg, "From: %s\r\n", from)
	}
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", encodeHeaderValue(input.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return base64.RawURLEncoding.EncodeToString([]byte(msg.String())), nil
}

func formatAddressHeader(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if err := validateHeader(name, value); err != nil {
		return "", err
	}
	addrs, err := mail.ParseAddressList(value)
	if err != nil {
		return value, nil
	}
	parts := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Name == "" {
			parts = append(parts, addr.Address)
		} else {
			parts = append(parts, addr.String())
		}
	}
	return strings.Join(parts, ", "), nil
}

func encodeHeaderValue(value string) string {
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("utf-8", value)
}

func isASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}

func validateHeader(name, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("gmail render raw %s header must not contain CR or LF", name)
	}
	return nil
}

func renderTemplate(pattern string, input any) (string, error) {
	tmpl, err := template.New("gmail_body").Option("missingkey=error").Parse(pattern)
	if err != nil {
		return "", fmt.Errorf("parse gmail body_template: %w", err)
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, input); err != nil {
		return "", fmt.Errorf("render gmail body_template: %w", err)
	}
	return b.String(), nil
}
