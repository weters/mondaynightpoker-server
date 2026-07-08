# Monday Night Poker - Architecture

This document describes the architecture of the Monday Night Poker open-source project.

## Services

```mermaid
flowchart LR
    FE["Vue.js SPA<br/>(mondaynightpoker-vue)"]
    BE["Go web API<br/>(mondaynightpoker-server)"]
    DB[("PostgreSQL")]

    FE -- "REST (auth, players, tables)" --> BE
    FE <-- "WebSocket (gameplay)" --> BE
    BE --> DB
```

### Front-end

The front-end is a single-page app built using Vue.js. The source code for the front-end is [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue).

Most of the communication happens to the back-end via REST. The gameplay itself is handled using websockets.

### Back-end

The back-end is a web API built using Go. Most of the API is RESTful, but the gameplay communicates over a websocket at `GET /table/{uuid}/ws`. Routing lives in [internal/mux](internal/mux) and is organized into three tiers: unauthenticated routes (health, sign-up, auth, account verification, password resets), authenticated routes guarded by a JWT bearer-token middleware, and admin routes that additionally require the player to be a site admin. Table-scoped routes live under a `/table/{uuid}` subrouter with its own middleware.

Authentication uses RSA-signed JWTs (see [internal/jwt](internal/jwt)) with a 30-day TTL and a refresh endpoint. Sign-ups are protected by Google reCAPTCHA v3.

### Database

The datastore is a PostgreSQL database. Connection handling and migrations are wrapped by the [pkg/db](pkg/db) package, which uses [golang-migrate/migrate](https://github.com/golang-migrate/migrate) under the hood. The migration files are found in the [./sql](./sql) directory. You can manually run the migrations using the following command:
```shell
$ go run ./cmd/migrate
```

If you want to migrate to a specific version, you can use the `-v` flag.
```shell
$ go run ./cmd/migrate -v 5
```

Note: migrations are automatically run when you start the web server

#### Database Design

Table | Description
--- | ---
`players` | Contains all of the players who sign up with Monday Night Poker. Tracks account status (`created`, `verified`, etc.), site-admin flag, and an argon2 password hash.
`tables` | A table can be thought of as a single game session. Tracks its creator and supports soft-deletion.
`players_tables` | Assigns players to a particular table. Holds the player's balance, table stake, per-table permissions (`can_start`, `can_restart`, `can_terminate`), and admin/blocked flags.
`players_tables_transactions` | A change log for every change made to the `players_tables.balance` column.
`games` | Keeps track of individual games played at a table. The final game log is stored as JSONB in the `data` column.
`player_tokens` | Used for use-once style tokens like when verifying an account or resetting a password.

```mermaid
erDiagram
    players ||--o{ players_tables : "sits at"
    tables ||--o{ players_tables : "seats"
    players ||--o{ tables : "creates"
    players_tables ||--o{ players_tables_transactions : "balance changes"
    games ||--o{ players_tables_transactions : "caused by"
    tables ||--o{ games : "hosts"
    players ||--o{ player_tokens : "issued"

    players {
        bigint id PK
        text email
        text display_name
        bool is_site_admin
        text password_hash
        player_status status
    }
    tables {
        text uuid PK
        text name
        bigint player_id FK "creator"
        timestamp deleted "soft delete"
    }
    players_tables {
        bigint id PK
        bigint player_id FK
        text table_uuid FK
        int balance
        int table_stake
        bool is_table_admin
        bool can_start
        bool can_restart
        bool can_terminate
        bool is_blocked
        bool active
    }
    games {
        bigint id PK
        text table_uuid FK
        bigint parent_id FK
        text game_type
        jsonb data
        timestamp ended
    }
    players_tables_transactions {
        bigint id PK
        bigint players_tables_id FK
        bigint game_id FK
        int adjustment
        int current_balance
    }
    player_tokens {
        text token PK
        bigint player_id FK
        text type "password reset or account verify"
        bool active
    }
```

## Code Layout

This codebase tries to adhere to the [Standard Go Project Layout](https://github.com/golang-standards/project-layout).

Below are the primary directories within the project

Dir | Description
--- | ---
[cmd/server](cmd/server) | The main web server
[cmd/migrate](cmd/migrate) | Standalone database migration runner (waits for the database to become available)
[cmd/admin](cmd/admin) | CLI admin tool; interactively creates a verified player and can promote them to site admin
[cmd/generate-config](cmd/generate-config) | Prints the default configuration as YAML for bootstrapping a `config.yaml`
[cmd/testbot](cmd/testbot) | A [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI bot that spins up (or takes over) a table full of bot players over REST + websockets, with an auto-pilot mode for exercising games end-to-end
[deployments](deployments) | Kubernetes configuration
[internal/config](internal/config) | Contains logic for loading and retrieving configuration (defaults → YAML file → `MNP_*` environment variables)
[internal/email](internal/email) | Handles sending emails and rendering email templates
[internal/jwt](internal/jwt) | Handles signing and verifying the JWT (RSA keys, 30-day TTL)
[internal/mux](internal/mux) | Contains the controller logic, middleware tiers, and the websocket endpoint
[internal/rng](internal/rng) | A small randomness abstraction (`Generator` interface) with a `crypto/rand`-backed implementation used for shuffling
[internal/util](internal/util) | Contains some miscellaneous helper functions (e.g., random display-name generation)
[pkg/db](pkg/db) | Wraps `database/sql` for PostgreSQL and runs golang-migrate migrations
[pkg/deck](pkg/deck) | Has the deck and card logic
[pkg/playable](pkg/playable) | Contains all the game logic and rules.
[pkg/room](pkg/room) | Responsible for managing the live game state.
[pkg/model](pkg/model) | The data-access layer. All queries to the PostgreSQL database are written here, organized into repositories (`Repositories` bundles the player, table, and game repos).
[pkg/mnptoken](pkg/mnptoken) | A small package for generating crypto-secure tokens (password resets, account confirmation)
[pkg/snapshot](pkg/snapshot) | A JSON snapshot-testing helper used by the game packages' tests (golden files under `testdata/`)
[sql](sql) | Contains the database migration files
[templates](templates) | Email templates

### Concepts

All gameplay is handled through websockets. When clients connect to the web service, we need a way of assigning them to the correct table and then handle the communication and gameplay.

The [room](pkg/room) package contains logic for managing clients and communication. The web service uses a single instance of `PitBoss`. The `PitBoss` listens for client connections and disconnections and takes appropriate action. When the first client connects for a table, the `PitBoss` creates a new `Dealer`. All subsequent users for that same table will join the same `Dealer`. When the last client for that table disconnects, the `PitBoss` removes the dealer.

**Important:** the `Dealer` has a primary run loop through which all state changes must happen to prevent race conditions. This is the most important thing to understand in the codebase and must not be broken.

The run loop itself never blocks on the database:

* Database *writes* are handed off to a dedicated `persistLoop()` goroutine via a `persist` channel (`enqueuePersist`). While a finished game's results are being persisted, a flag gates the next game start so it always reads post-adjustment balances.
* Database *reads* (roster and permission lookups) run on background goroutines and re-enter the run loop via `tryExecInRunLoop`; a generation counter discards stale roster fetches.

The `Dealer` handles all client messages in the `ReceivedMessage()` method. Each message is a `playable.PayloadIn`. These messages have an `Action` field that determines what action the dealer should take. Some common actions are to `createGame` or `terminateGame`. Any action not handled by the `Dealer` is sent to the active game being played.

The `createGame` action looks up a `GameFactory` in the [room/gamefactory](pkg/room/gamefactory) registry by the game's slug (`bourre`, `seven-card`, `pass-the-poop`, `little-l`, `acey-deucey`, `texas-hold-em`, `guts`) and asks it to construct a game that implements the `playable.Playable` interface.

```mermaid
classDiagram
    class PitBoss {
        -dealers map[string]*Dealer
        -connect chan *Client
        -disconnect chan *Client
        +StartShift()
        +ClientConnected(c *Client)
        +ClientDisconnected(c *Client)
    }
    class Dealer {
        -clients map[*Client]bool
        -game playable.Playable
        -execInRunLoop chan func()
        -persist chan func()
        +StartShift()
        +ReceivedMessage(c *Client, msg *PayloadIn)
        +AddClient(c *Client)
        +RemoveClient(c *Client) bool
        +EndShift()
        -runLoop()
        -persistLoop()
    }
    class Client {
        +Conn *websocket.Conn
        -send chan interface
        +Send(msg) bool
        +ReceivedMessage(msg *PayloadIn)
        +Disconnect(reason)
    }
    class TableStore {
        <<interface>>
        +GetPlayers()
        +GetActivePlayersShifted()
        +CreateGame()
        +EndGame()
        +SavePlayerTable()
    }
    class GameFactory {
        <<interface>>
        +CreateGame(logger, players, additionalData) Playable
        +Details(additionalData) (name, ante)
    }
    class Playable {
        <<interface>>
    }

    PitBoss "1" o-- "*" Dealer : by table UUID
    Dealer "1" o-- "*" Client
    Dealer --> "0..1" Playable : active game
    Dealer --> TableStore : persists via
    Dealer ..> GameFactory : createGame
    GameFactory ..> Playable : constructs
    Client --> Dealer : forwards messages
```

The [playable](pkg/playable) package contains the actual game logic.

All games must implement the `Playable` interface. There are only five methods that must be implemented:

1. `Name() string` which returns the name of the game
2. `LogChan() <-chan []*LogMessage` returns a channel that the game will send log messages to that should be passed to the client
3. `Action(playerID int64, message *PayloadIn) (playerResponse *Response, updateState bool, err error)` handles a user action. For example, if the user places a bet, this is the method that actually does that action.
4. `GetPlayerState(playerID int64) (*Response, error)` returns the current state of the game for that player.
5. `GetEndOfGameDetails() (gameOverDetails *GameOverDetails, isGameOver bool)` will be called periodically to see if the game is over. If the game is over, details such as the final game log and player balance adjustments is returned.

That's it. Implement those five methods, and you have a game that can be handled by Monday Night Poker. In practice, games embed `*playable.Core`, which provides the log channel plumbing (`LogChan()`, `SendLogMessage()`) and a default tick interval, so each game only writes the logic that makes it unique.

Games may implement the `Tickable` interface (`Interval() time.Duration` and `Tick() (bool, error)`). While you can run a game using only the `Playable` interface, your game will be reactive. That means in order for the game state to change, a player must progress it. `Tickable` allows the game to periodically update the state without user interaction. This is typically used to delay the game to allow users to process what's happening, e.g., turn one card at a time instead of flopping all 3.

Games may also implement `RulesProvider` (`Rules() []RuleSection`) to send the client a formatted description of the game's rules.

The games currently implemented are:

* [aceydeucey](pkg/playable/aceydeucey) — Acey Deucey
* [bourre](pkg/playable/bourre) — Bourré
* [guts](pkg/playable/guts) — Guts
* [passthepoop](pkg/playable/passthepoop) — Pass the Poop (Standard, Pairs, and Diarrhea editions)
* [poker/littlel](pkg/playable/poker/littlel) — Little L
* [poker/sevencard](pkg/playable/poker/sevencard) — Seven-card games. A `Variant` interface allows many rule sets on the same engine: Stud, Baseball, Follow the Queen, Low Card Wild, High Chicago, and more.
* [poker/texasholdem](pkg/playable/poker/texasholdem) — Texas Hold'em (Standard, Pineapple, and Lazy Pineapple variants)

The poker games share several helper packages:

* [poker/handanalyzer](pkg/playable/poker/handanalyzer) analyzes a poker hand. The `HandAnalyzer` struct is the work-horse: give it a hand size and cards (wild cards supported) and it determines the best hand and its strength.
* [poker/potmanager](pkg/playable/poker/potmanager) manages betting rounds, blinds, side pots, all-ins, and payouts.
* [poker/action](pkg/playable/poker/action) defines the common betting actions (check, call, bet, raise, fold, etc.).

```mermaid
classDiagram
    class Playable {
        <<interface>>
        +Name() string
        +LogChan() chan LogMessage
        +Action(playerID, message) (Response, bool, error)
        +GetPlayerState(playerID) Response
        +GetEndOfGameDetails() (GameOverDetails, bool)
    }
    class Tickable {
        <<interface>>
        +Interval() time.Duration
        +Tick() (bool, error)
    }
    class RulesProvider {
        <<interface>>
        +Rules() []RuleSection
    }
    class Core {
        +LogChan()
        +SendLogMessage()
        +Interval() time.Duration
    }
    class AceyDeucey
    class Bourre
    class Guts
    class PassThePoop
    class LittleL
    class SevenCard
    class TexasHoldEm
    class Variant {
        <<interface>>
        seven-card rule sets
    }
    class HandAnalyzer {
        +GetStrength()
    }
    class PotManager {
        betting, side pots, payouts
    }

    Playable <|.. AceyDeucey
    Playable <|.. Bourre
    Playable <|.. Guts
    Playable <|.. PassThePoop
    Playable <|.. LittleL
    Playable <|.. SevenCard
    Playable <|.. TexasHoldEm
    Tickable <|.. PassThePoop
    Tickable <|.. LittleL
    Tickable <|.. SevenCard
    Tickable <|.. TexasHoldEm
    Core <|-- AceyDeucey : embeds
    Core <|-- Bourre : embeds
    Core <|-- Guts : embeds
    Core <|-- PassThePoop : embeds
    Core <|-- LittleL : embeds
    Core <|-- SevenCard : embeds
    Core <|-- TexasHoldEm : embeds
    SevenCard o-- Variant
    LittleL ..> HandAnalyzer
    SevenCard ..> HandAnalyzer
    TexasHoldEm ..> HandAnalyzer
    LittleL ..> PotManager
    SevenCard ..> PotManager
    TexasHoldEm ..> PotManager
```

The [deck](pkg/deck) package provides capabilities around cards and decks.

The games all make use of a `Deck`, although they don't have to. Decks are comprised of many `Card` objects. The cards may be assembled into a `Hand`. Shuffling uses the `crypto/rand`-backed generator from [internal/rng](internal/rng) (tests can swap in a seeded generator). A five-suit deck (`NewFiveSuit()`) is available for games that need it, and cards carry bit flags and wild-card metadata for the wild-card poker variants.

```mermaid
classDiagram
    class Deck {
        +Cards []*Card
        +Shuffle()
        +Draw() *Card
        +CardsLeft() int
        +SetSeed(seed)
    }
    class Card {
        +Rank int
        +Suit Suit
        +Equal(c) bool
        +AceLowRank() int
        +SetWildRank(rank)
    }
    class Hand {
        +AddCard(c)
        +HasCard(c) bool
        +Discard(c)
    }
    class Generator {
        <<interface>>
        +Intn(n) int
    }

    Deck "1" o-- "*" Card
    Hand "1" o-- "*" Card
    Deck --> Generator : shuffles with
```

## Configuration

Configuration is resolved in three layers, each overriding the last: hardcoded defaults → a YAML file (`$MNP_CONFIG_FILE`, falling back to `./config.yaml`) → `MNP_`-prefixed environment variables (e.g., `MNP_DATABASE_DSN`, `MNP_RECAPTCHA_SECRET`). Run `go run ./cmd/generate-config` to print the defaults as a starter YAML file. See the [README](README.md#configuration) for the full list of settings.

## Deployment

The server is built into a two-stage Docker image (Go build → Alpine runtime containing the server binary, the `sql/` migrations, and the email `templates/`) and deployed into a Kubernetes cluster. The manifests can be found in the [deployments](deployments) directory. The container exposes port 5080 and uses `/health` for readiness and liveness probes.

The k8s deployment requires two additional secrets: one for runtime credentials and one for the JWT signing keys (mounted at `/app/.keys`).

```yaml
apiVersion: v1
metadata:
  labels:
    app: mondaynightpoker-server
  name: mondaynightpoker-server-config
stringData:
  email_password: ""
  pg_dsn: ""
  recaptcha_secret: ""
kind: Secret

---

apiVersion: v1
kind: Secret
metadata:
  name: mondaynightpoker-server-keys
  labels:
    app: mondaynightpoker-server
stringData:
  public.pem: |
    -----BEGIN PUBLIC KEY-----
    ...
    -----END PUBLIC KEY-----
  private.key: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```

## Future Enhancements

* **Horizontal scaling.** Right now this service cannot be scaled horizontally because of websockets. Game state is kept in-memory. Before we can scale the back-end, we need to ensure that all users are stickied to the server hosting the game.
* **WebSocket authentication.** The websocket currently authenticates via an `?access_token=` query parameter; this should move to a header or first-message handshake (see [docs/DEEP_MODERNIZATION.md](docs/DEEP_MODERNIZATION.md)).
* **Service layer.** Some business rules (password hashing, stats grouping, seat rotation) live alongside the SQL in `pkg/model`; they could be extracted into a service layer if the domain grows.
