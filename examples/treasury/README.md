# Treasury Example

This example is a larger Gapi application that models a tenant-scoped treasury control plane.

It demonstrates:

- Bearer authentication with `gapi.Dep`.
- Tenant isolation and role checks.
- Account creation and balance tracking.
- Idempotent transfer creation through the `Idempotency-Key` header.
- Transfer risk scoring and state transitions.
- Double-entry ledger output.
- Audit trail and platform event stream.
- JSON responses, `gapi.Response[T]`, CSV via `gapi.Text`, and SSE via `gapi.SSE`.
- OpenAPI metadata with tags, summaries, validation, and bearer security.

Run:

```bash
go run ./examples/treasury
```

Open:

```text
http://localhost:8080/docs
http://localhost:8080/openapi.json
```

Demo tenant:

```text
tnt_demo
```

Tokens:

```text
owner:   sentinel_owner_token
analyst: sentinel_analyst_token
```

Try:

```bash
curl -H "Authorization: Bearer sentinel_owner_token" \
  http://localhost:8080/api/v1/me

curl -H "Authorization: Bearer sentinel_analyst_token" \
  http://localhost:8080/api/v1/tenants/tnt_demo/accounts
```
