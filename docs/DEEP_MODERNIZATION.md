# Deep Modernization Plan

Deferred architectural work identified during the July 2026 architecture review of
`mondaynightpoker-server` and `mondaynightpoker-vue`. The correctness fixes from that
review (realtime-layer races, WebSocket origin validation, the frontend game registry,
WS service extraction, and Vitest infrastructure) already landed; the items below are
the larger structural investments, roughly ordered by value.

## Backend

### 1. Repository interfaces + dependency injection
The codebase is wired through package-level singletons, which blocks mocking and forces
every model/mux test onto a live database:

- `pkg/db/db.go` — global `*sql.DB` with a racy lazy init (`Instance()` does an
  unsynchronized nil check before `LoadInstance()`, which panics on failure).
- `internal/config` — mutex-guarded singleton that panics on load failure; packages read
  it at init time (`pkg/room/pendinggame.go` captures `StartGameDelay` at package init,
  an ordering hazard).
- `internal/jwt` — package-global key pair loaded by `LoadKeys()`.
- `internal/email` — global default client config.

Plan: construct the DB handle, config, JWT signer, and email client once in `main()`,
define per-aggregate repository interfaces (`PlayerRepository`, `TableRepository`, …),
and inject them into `mux`/`room`. Return errors instead of panicking outside `main`.

### 2. Split the `pkg/model` active-record god package
`model/player.go` (~760 lines) mixes SQL, validation, argon2 hashing, token logic, and
stats aggregation; `Table.GetActivePlayersShifted` embeds game-rotation business rules in
the persistence layer. Once repositories exist (item 1), move non-SQL rules into a
service layer and keep the persistence layer mechanical.

### 3. Move blocking DB I/O out of the Dealer run loop
The per-table run loop serializes all gameplay, yet performs synchronous DB calls:
`sendPlayerData` → `table.GetPlayers` (dealer.go), `endGame` → `CreateGame`/`EndGame`,
`createGame` → `GetActivePlayersShifted`, and the `ReceivedMessage` closures
(`tableAdmin`, `tableStake`, `playerStatus`) do `GetPlayerTable`/`Save` inside the loop.
A slow query freezes every player's input on that table. Plan: fetch data before
entering the loop and pass results in; make post-game persistence async with retry.

Related (same bug class, found during the review): `Dealer.ReceivedMessage` reads
`d.game != nil` on the WebSocket read goroutine (dealer.go, `createGame` case) even
though game state is owned by the run loop. Harmless today (it only picks which
permission to check) but should move inside the run loop when this item is tackled.

### 4. Finish the V1→V2 game factory migration
`gamefactory` has parallel `CreateGame`/`CreateGameV2` interfaces; the dealer type-asserts
`gamefactory.V2` in the hot path, `little-l`/`texas-hold-em` panic if the V1 path is hit,
and `sevencard` carries a deprecated `simplePlayer` shim (sevencard.go). Plan: make the
`[]*model.PlayerTable` signature the only interface, delete V1 constructors and the shim.

### 5. Extract a shared BaseGame
Every game copy-pastes the same plumbing: a `logChan chan []*playable.LogMessage`,
`sendLogMessage`/`newLogMessage` helpers, and a mutable package-global `var seed int64`
test hook (e.g. `passthepoop/game.go`, `littlel/littlel.go`). Plan: an embeddable
`BaseGame` providing the log channel + helpers, and a shared deterministic-seed test
utility in place of per-package globals.

### 6. JWT expiry + refresh flow
`jwt.Sign` sets `IssuedAt` but no `ExpiresAt` — tokens never expire and cannot be
revoked. The frontend stores the token in `localStorage` and authenticates the WebSocket
via `?access_token=` in the URL (visible to access logs/proxies). Plan: short-lived
access tokens + refresh endpoint, WS auth via first-message or `Sec-WebSocket-Protocol`
header instead of the query string, and an FE refresh/re-auth path so introducing expiry
doesn't silently log everyone out.

### 7. Uniform per-game action envelope
Three incompatible client→server action encodings exist today:
- poker family: the action constant is the WS `action` verb (`bet`, `fold`, `discard`…)
- acey-deucey / pass-the-poop: `action: "execute"` with a stringified integer enum in
  `subject`
- bourre / guts: bespoke verbs (`playCard`, `discard`, `decide`, `trade`)

Plan: one envelope (e.g. `action: "gameAction", subject: <actionID>, payload: {…}`),
with each game exposing its available actions in player state the way poker already
does (`{id, name}` objects). Migrate game-by-game; the FE `webSocketSend` action is now
the single choke point, which makes this tractable.

## Frontend

### 8. Vuex → Pinia
Vuex 4 is in maintenance mode. The store surface is now small (root store + six
getter-only modules registered through `src/games/index.js`), so a Pinia migration is
mostly mechanical. Do after the action-envelope work to avoid rewriting per-game modules
twice.

### 9. Route-level code splitting + explicit auth meta
`main.js` statically imports every route component (one large bundle — the build warns
about chunk size) and treats routes as protected unless they opt out via
`meta.protected: false`, which is implicit and easy to get wrong. Plan: `router.js` with
`() => import()` lazy routes and explicit `meta.requiresAuth: true`.

### 10. Tooling: ESLint 9 + stronger ruleset, Sass `@use`
- ESLint 8 is EOL; the config uses `plugin:vue/vue3-essential` (the weakest preset) and
  disables `vue/multi-word-component-names` (which is why `Error.vue`/`Boolean.vue`
  shadow globals). Plan: flat config, `vue3-recommended`, rename shadowing components.
- 30+ files use deprecated Sass `@import '../variables.scss'` with relative paths;
  deprecations are silenced in `vite.config.js`. Plan: `@use '@/variables' as *` via
  `css.preprocessorOptions.scss.additionalData`, then remove the per-file imports and
  the `silenceDeprecations` entry.

### 11. Remove the mitt event bus
`bus.js` remains for two events: `error` (webSocket → PokerTable, parallel to the store
error channel) and `edit-player` (PokerTablePlayer dropdown coordination). Plan: route
errors exclusively through the store action, and coordinate the player dropdown via
store state or provide/inject; then delete the bus.

### 12. Decompose `SevenCardParticipant.vue`
At ~500 lines it is the largest component and hard-codes two variant effect systems
(Chiggs mushroom/antidote, Coupons & Clippings BOGO) as ~14 reactive flags with manual
timeouts. Plan: move variant/effect derivation into store getters or a composable and
make the participant component presentational.

## Security hygiene

- The root `config.yaml` (gitignored, local-only) holds a live recaptcha secret and an
  email password in plaintext. Since envconfig already supports
  `MNP_RECAPTCHA_SECRET`/`MNP_EMAIL_PASSWORD`, consider moving the secrets to the
  environment or a secret store so a stray commit or backup can't leak them.
