# Helper And Gmail Setup

`apitools/helper` exposes small helper contracts that downstream authoring tools
and trusted runtimes can share without importing a workflow executor.

The first concrete helper package is `helper/gmailmsg`.

## Boundary

`apitools` owns:

- public helper descriptors;
- pure payload-shaping helper implementations such as `gmail.render_raw`.

`apitools` does not own:

- UWS workflow execution;
- Gmail API calls;
- runtime account selection;
- secret storage;
- OAuth credential parsing, consent, or token exchange;
- trusted-runner approval or package handoff.

Trusted runtimes such as `../udon` decide when helper functions are registered
and when OAuth material is resolved for execution.

## Fnct Helper Catalog

The parent package `github.com/OpenUdon/apitools/helper` exposes descriptors
and registration hooks:

```go
specs := helper.FunctionSpecs()
spec, ok := helper.LookupFunctionSpec("gmail.render_raw")
helper.RegisterFnctHelpers(registrar)
_, _ = specs, spec
_ = ok
```

`RegisterFnctHelpers` expects a minimal runtime registrar:

```go
type FnctRegistrar interface {
	AddStubMap(string, any)
}
```

Runtimes own the actual registration point. The helper catalog only provides
names, function contracts, and Go functions.

## `gmail.render_raw`

`gmail.render_raw` turns a request-body object into the base64url raw message
string expected by Gmail `users.messages.send`.

Input shape:

```json
{
  "to": "me@example.com",
  "subject": "Weather report",
  "body_template": "Weather report:\n\n{{.}}",
  "input": {
    "summary": "Toronto weather..."
  }
}
```

Fields:

| Field | Required | Meaning |
|---|---:|---|
| `to` | yes | Recipient email address. |
| `from` | no | Optional sender header. |
| `subject` | yes | Email subject. |
| `body` | one of `body` or `body_template` | Literal plain-text body. |
| `body_template` | one of `body` or `body_template` | Go `text/template` body rendered with `input` as dot. |
| `input` | no | Template data. |

Output:

```json
{
  "received_body": "<gmail raw message string>"
}
```

In OpenUdon/Udon workflows this is commonly authored as a `fnct` operation,
then passed to Gmail send as `raw`:

```hcl
operation "render_weather_report" {
  request {
    body {
      to            = { __dollar__expr = "variables.inputs.recipient_email" }
      subject       = "Weather report"
      body_template = "Weather report:\n\n{{.}}"
      input         = { __dollar__expr = "fetch_weather.received_body" }
    }
  }

  extensions {
    x-uws-operation-profile = "uws.runtime.1.0"
    x-uws-runtime {
      type     = "fnct"
      function = "gmail.render_raw"
    }
  }
}
```

`gmail.render_raw` does not call Gmail and does not resolve credentials.

## Google OAuth Setup

For Gmail send workflows, the runtime needs a Google OAuth2 credential with the
Gmail send scope:

```text
https://www.googleapis.com/auth/gmail.send
```

OAuth consent and token exchange are runtime responsibilities, not apitools
helper behavior. Use Udon's trusted operator CLI with a Google Desktop OAuth
client:

```bash
export GOOGLE_CLIENT_SECRET='...'

udon oauth google login \
  --client-id 395074190883-pl1v83a1v9q2o41m9p1kqfbpu720ufbo.apps.googleusercontent.com \
  --scope https://www.googleapis.com/auth/gmail.send \
  --output ./google-oauth.hcl
```

The command binds only to loopback and chooses an ephemeral port by default:

```text
127.0.0.1:0
```

The actual redirect is `http://127.0.0.1:<port>/oauth2/callback`. Authorization
uses a fresh state value and PKCE verifier. The refresh token is never printed;
it is written only to the requested new file, which must not already exist and
is created with mode `0600`. Keep that file out of version control. Use
`--listen 127.0.0.1:8765` only when the OAuth client requires a fixed loopback
port.

## Data File For Udon

Keep non-secret workflow inputs in a reviewed data file:

```hcl
inputs {
  recipient_email = "ENVIRONMENT:RECIPIENT_EMAIL"
}

```

Pass the private OAuth file separately when running Udon:

```bash
export RECIPIENT_EMAIL='me@example.com'
export GOOGLE_CLIENT_SECRET='...'

udon --workdir . --workflow workflows/workflow.hcl \
  --datafile expected/data.hcl \
  --datafile ./google-oauth.hcl
```

Udon resolves `ENVIRONMENT` markers at execution time. If an environment
variable is unset, execution fails before provider calls.

`UDON_CREDENTIAL_GOOGLEOAUTH2` still takes priority when present. Runtime data
files may supply an existing access token or refresh token; Udon deliberately
does not exchange a manually pasted authorization code without the command's
PKCE verifier.

## Common Issues

- **Redirect URI mismatch**: use a Google Desktop OAuth client for dynamic
  loopback redirects, or choose an explicitly configured loopback port with
  `--listen`.
- **No refresh token returned**: Google may not return a refresh token if the
  user already granted the app. Revoke the app grant in the Google account,
  then rerun the command.
- **Wrong scope**: Gmail send needs
  `https://www.googleapis.com/auth/gmail.send`.
- **Output already exists**: choose a new output path or move the old private
  file deliberately; the command never overwrites credential files.
- **Do not commit secrets**: `google-oauth.hcl` contains a refresh token and
  must remain private.
