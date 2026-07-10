# mondaynightpoker-server

[![CI/CD](https://github.com/weters/mondaynightpoker-server/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/weters/mondaynightpoker-server/actions/workflows/ci.yaml)

mondaynightpoker-server is the backend application for the Monday Night Poker site (see: [mondaynight.bid](https://mondaynight.bid)). The front-end code can be found at [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue).

For more information about the architecture of this open-source project, see the [architecture](architecture.md) document.

## Supported Card Games

* Acey Deucey
* Bourré
* Guts
* Pass the Poop
  * Standard Edition
  * Diarrhea Edition
  * Pairs Edition
* Poker
  * Pot-Limit Texas Hold'em
  * Little L
  * Seven-card games
    * Seven-Card Stud
    * Follow the Queen
    * Baseball
    * High Chicago
    * Low Card Wild
    * Coupons and Clippings
    * 7 Card Chiggs

## Getting Started

### Prerequisites

1. [Go 1.24+](https://golang.org/dl/)
2. [golangci-lint v2](https://golangci-lint.run/docs/install/)
3. [Docker](https://www.docker.com/products/docker-desktop)
4. Google [reCAPTCHA v3](https://www.google.com/u/1/recaptcha/admin/create) Secret

### Development

1. Create the dev database

```
$ make dev-database
```
    
2. Create your public and private keys for JWT signing

```
$ make keys
```
    
3. Make an admin user

```
$ go run ./cmd/admin -c user
```

4. Run the server

```
$ MNP_RECAPTCHA_SECRET=X go run ./cmd/server
```
    
5. Verify the server is running

```
$ curl http://localhost:5080/health
```
    
6. Start the Vue.js front-end. Repo can be found at [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue)

### Configuration

The service can be configured through two methods:

1. **YAML Configuration:** By default, the service will look for `config.yaml`. You can also change the filename by setting a `MNP_CONFIG_FILE` environment variable.
2. **Environment Variables:** All configuration settings can be set by environment variables. Every variable is prefixed by `MNP_` and `camelCase` is transformed to `SNAKE_CASE`. Example, `jwt.publicKey` will become `MNP_JWT_PUBLIC_KEY`.

Any environment variables take precedence over values defined in YAML. The default configuration values are defined below.

```yaml
host: https://mondaynight.bid
# Browser origins allowed for both CORS (REST) and the WebSocket upgrade.
# When empty, only `host` is permitted.
allowedOrigins:
  - https://mondaynight.bid
  - https://beta.mondaynight.bid
log:
  disableAccessLogs: false
  level: info
database:
  dsn: postgres://postgres@localhost:5432/postgres?sslmode=disable
  migrationsPath: ./sql
jwt:
  publicKey: .keys/public.pem
  privateKey: .keys/private.key
recaptchaSecret: '-'
startGameDelay: 10
playerCreateDelay: 60
email:
  from: Monday Night Poker <no-reply@mondaynight.bid>
  sender: no-reply@mondaynight.bid
  username: dealer@mondaynight.bid
  password: ""
  host: mail.privateemail.com:587
  templatesDir: templates
  disable: false
```

> **Note on `allowedOrigins`:** this single list is enforced by both the CORS
> layer (REST) and the WebSocket upgrade check, so the two can never disagree
> about which browser origins are permitted. When developing locally against a
> front-end on a different origin (e.g. `http://localhost:8080`), add that origin
> to `allowedOrigins` or set `MNP_ALLOWED_ORIGINS=http://localhost:8080` — a
> missing entry now blocks REST as well as the WebSocket.

You can generate a YAML file with the defaults by running:

```shell
$ go run ./cmd/generate-config
```