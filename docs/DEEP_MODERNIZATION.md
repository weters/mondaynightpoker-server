# Deep Modernization Plan

The July 2026 modernization pass completed nearly everything this document
originally tracked: the V1→V2 game-factory migration, the shared `playable.Core`
game plumbing, repositories + dependency injection (no more global `sql.DB`,
config singleton, or JWT key globals), moving all database I/O off the dealer
run loop, 30-day JWTs with a refresh endpoint and frontend refresh/401 handling,
the unified string-verb action envelope, the Pinia migration, route-level code
splitting with explicit auth meta, ESLint 9 (flat config, vue3-recommended,
zero-warning gate), the Sass `@use` migration, mitt-bus removal, and the
SevenCardParticipant decomposition.

## Remaining items

### WebSocket auth still rides the query string
The WebSocket authenticates via `?access_token=<jwt>` in the URL
(`internal/mux/mux.go` authMiddleware reads `FormValue`), which can leak tokens
into access logs and proxies. Token expiry and rotation shrink the exposure
window, but the transport should eventually move to a `Sec-WebSocket-Protocol`
header or a first-message auth handshake, paired with a frontend change in
`src/webSocket.js`. A related residual: a WebSocket-only tab whose token expires
will reconnect-loop at the 30-second backoff cap until any REST call returns 401
and forces a re-login.

### Enforce token expiry for legacy tokens
`jwt.Signer.ValidUserID` still accepts tokens without an `exp` claim so that
sessions issued before expiry existed keep working; they rotate onto expiring
tokens the next time the client boots. Once those have aged out (~90 days after
deploying the refresh flow), add `jwtgo.WithExpirationRequired()` to the parser
options in `internal/jwt/jwt.go`.

### Split `pkg/model` business rules out of the repositories
The repository layer now owns all SQL behind `model.Repositories`, but a few
business rules still live beside the queries: argon2 hashing inside
`PlayerRepo.CreatePlayer`/`ResetPassword`, stats grouping in
`PlayerRepo.GetPlayerStats`, and the rotation math in
`TableRepo.GetActivePlayersShifted`. If the domain grows, extract these into a
service layer; at the current size they are fine where they are.

### Security hygiene
The root `config.yaml` (gitignored, local-only) holds a live recaptcha secret
and an email password in plaintext. envconfig already supports
`MNP_RECAPTCHA_SECRET`/`MNP_EMAIL_PASSWORD`; consider moving the secrets to the
environment or a secret store so a stray commit or backup cannot leak them.
