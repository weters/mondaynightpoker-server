# Monday Night Poker - Architecture

This document describes the architecture of the Monday Night Poker open-source project.

## Services

### Front-end

The front-end is a single-page app built using Vue.js. The source code for the front-end is [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue).

Most of the communication happens to the back-end via REST. The gameplay itself is handled using websockets.

### Back-end

The back-end is a web API built using Golang. Most of the API is RESTful, but the gameplay communicates using websockets.

### Database

The datastore is a PostgreSQL database. Migrations are handled using [golang-migrate/migrate](https://github.com/golang-migrate/migrate). The migration files are found in the [./sql](./sql) directory. You can manually run the migrations using the following command:
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
`players` | Contains all of the players who sign up with Monday Night Poker.
`tables` | A table can be thought of as a single game session.
`players_tables` | Assigns players to a particular table. Balances are kept within this table.
`players_tables_transactions` | A change log for every change made to the `players_tables.balance` column.
`games` | Keeps track of individual games played at a table.
`player_tokens` | Used for use-once style tokens like when verifying an account or resetting a password.

![Database Design](assets/tables.png)

## Code Layout

This codebase tries to adhere to the [Standard Go Project Layout](https://github.com/golang-standards/project-layout).

Below are the primary directories within the project

Dir | Description
--- | ---
[cmd](cmd) | Contains command-line applications
[deployments](deployments) | Kubernetes configuration
[internal/config](internal/config) | Contains logic for loading and retrieving configuration
[internal/email](internal/email) | Handles sending emails and rendering email templates
[internal/jwt](internal/jwt) | Handles signing and verifying the JWT
[internal/mux](internal/mux) | Contains the controller logic
[internal/util](internal/util) | Contains some miscellaneous helper functions
[pkg/deck](pkg/deck) | Has the deck and card logic
[pkg/playable](pkg/playable) | Contains all the game logic and rules.
[pkg/room](pkg/room) | Responsibly for managing the live game state.
[pkg/table](pkg/table) | This is essentially the database model. All queries to the PostgreSQL database are written here.
[pkg/token](pkg/token) | A small package for generate a crypto-secure token.
[sql](sql) | Contains the database migration files
[templates](templates) | Email templates

### Concepts

All gameplay is handled through websockets. When clients connect to the web service, we need a way of assigning them to the correct table and then handle the communication and gameplay.

The [room](pkg/room) package contains logic for managing clients and communication. The web service uses a single instance of `PitBoss`. The `PitBoss` listens for client connections and disconnections and takes appropriate action. When the first client connects for a table, the `PitBoss` creates a new `Dealer`. All subsequent users for that same table will join the same `Dealer`. When the last client for tha table disconnects, the `PitBoss` removes the dealer.

**Important:** the `Dealer` has a primary run loop through which all state changes must happen to prevent race conditions. This is the most important thing to understand in the codebase and must not be broken.

The `Dealer` handles all client messages in the `ReceivedMessage()` method. Each message is a `playable.PayloadIn`. These messages have an `Action` field that determines what action the dealer should take. Some common actions are to `createGame` or `terminateGame`. Any action not handled by the `Dealer` is sent to the active game being played.

The `createGame` action attempts to create a game that implements the `playable.Playable` interface.

![Room UML](assets/uml_room.png)

The [playable](pkg/playable) package contains the actual game logic.

All games must implement the `Playable` interface. There are only five methods that must be implemented:

1. `Name() string` which returns the name of the game
2. `LogChan() <-chan []*LogMessage` returns a channel that the game will send log messages to that should be passed to the client
3. `Action(playerID int64, message *PayloadIn) (playerResponse *Response, updateState bool, err error)` handles a user action. For example, if the user places a bet, this is the method that actually does that action.
4. `GetPlayerState(playerID int64) (*Response, error)` returns the current state of the game for that player.
5. `GetEndOfGameDetails() (gameOverDetails *GameOverDetails, isGameOver bool)` will be called periodically to see if the game is over. If the game is over, details such as the final game log and player balance adjustments is returned.

That's it. Implement those five methods, and you have a game that can be handled by Monday Night Poker.

Games may implement the `Tickable` interface. While you can run a game using only the `Playable` interface, your game will be reactive. That means in order for the game state to change, a player must progress it. The `Tickable` allows the game to periodically update the state without user interaction. This is typically used to delay the game to allow users to process what's happening, e.g., turn one-card at-a-time instead of flopping all 3.

The [handanalyzer](pkg/playable/poker/handanalyzer) package provides capabilities for analyzing a poker hand. The `HandAnalzyer` struct is the work-horse.

The games found in the [pkg/playable/poker](pkg/playable/poker) package use `HandAnalyzer` to analyze the hands. Below you can see a diagram of how Seven Card Poker and its variants are implemented.

![Playable UML](assets/uml_playable.png)

The [deck](pkg/deck) package provides capabilities around cards and decks.

The games all make use of a `Deck`, although they don't have to. Decks are comprised of many `Card` objects. The cards may be assembled into a `Hand`. This is all fairly straight-forward.

![Deck UML](assets/uml_deck.png)

## Deployment

The server is deployed into a Kubernetes cluster. The deployments can be found in the [deployments](deployments) directory.

The k8s deployment requires two additional resources: secrets.

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

Right now this service cannot be scaled horizontally because of websockets. Game state is kept in-memory. Before we can scale the back-end, we need to ensure that all users are stickied to the server hosting the game.
