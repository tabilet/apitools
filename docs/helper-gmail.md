# Helper And Gmail Setup

`apitools/helper` exposes small helper contracts that downstream authoring tools
and trusted runtimes can share without importing a workflow executor.

The first concrete helper package is `helper/gmailmsg`.

## Boundary

`apitools` owns:

- public helper descriptors;
- pure payload-shaping helper implementations such as `gmail.render_raw`;
- narrow OAuth2 bootstrap shapes and local operator CLI utilities for Google
  OAuth setup.

`apitools` does not own:

- UWS workflow execution;
- Gmail API calls;
- runtime account selection;
- secret storage;
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

Use the local operator CLI to get a refresh token:

```bash
export GOOGLE_CLIENT_SECRET='...'

go run ./cmd/apitools oauth google login \
  --client-id 395074190883-pl1v83a1v9q2o41m9p1kqfbpu720ufbo.apps.googleusercontent.com \
  --scope https://www.googleapis.com/auth/gmail.send
```

By default the command listens on:

```text
127.0.0.1:8765
```

and uses this redirect URL:

```text
http://127.0.0.1:8765/oauth2/callback
```

Add that URL to the OAuth client's authorized redirect URIs in Google Cloud
Console before running the command.

The command prints:

- the Google consent URL;
- an `export GOOGLE_REFRESH_TOKEN='...'` command;
- a marker-based `data.hcl` snippet.

It does not write files and does not call Gmail APIs.

## Data File For Udon

Use environment markers in `expected/data.hcl` so the reviewed package contains
names of environment variables, not plaintext secrets:

```hcl
inputs {
  recipient_email = "ENVIRONMENT:RECIPIENT_EMAIL"
}

credentials {
  googleOAuth2 {
    client_id     = "395074190883-pl1v83a1v9q2o41m9p1kqfbpu720ufbo.apps.googleusercontent.com"
    client_secret = "ENVIRONMENT:GOOGLE_CLIENT_SECRET"
    refresh_token = "ENVIRONMENT:GOOGLE_REFRESH_TOKEN"
  }
}
```

Then run Udon with those environment variables set:

```bash
export RECIPIENT_EMAIL='me@example.com'
export GOOGLE_CLIENT_SECRET='...'
export GOOGLE_REFRESH_TOKEN='...'

udon --workdir . --workflow workflows/workflow.hcl --datafile expected/data.hcl
```

Udon resolves `ENVIRONMENT` markers at execution time. If an environment
variable is unset, execution fails before provider calls.

`UDON_CREDENTIAL_GOOGLEOAUTH2` still takes priority when present. The data-file
OAuth block is a fallback for direct Udon execution.

## Manual Code Exchange

If you already have an authorization code, exchange it without starting the
local callback server:

```bash
export GOOGLE_CLIENT_SECRET='...'

go run ./cmd/apitools oauth google login \
  --client-id 395074190883-pl1v83a1v9q2o41m9p1kqfbpu720ufbo.apps.googleusercontent.com \
  --scope https://www.googleapis.com/auth/gmail.send \
  --code '<authorization-code>' \
  --redirect-url '<redirect-url-used-to-create-the-code>'
```

The redirect URL must exactly match the one used when Google issued the code.

## Common Issues

- **Redirect URI mismatch**: add
  `http://127.0.0.1:8765/oauth2/callback` to the OAuth client, or pass
  `--redirect-url` and configure that exact URL.
- **No refresh token returned**: Google may not return a refresh token if the
  user already granted the app. Revoke the app grant in the Google account,
  then rerun the command.
- **Wrong scope**: Gmail send needs
  `https://www.googleapis.com/auth/gmail.send`.
- **Do not commit secrets**: commit marker names such as
  `ENVIRONMENT:GOOGLE_REFRESH_TOKEN`, not token values.
