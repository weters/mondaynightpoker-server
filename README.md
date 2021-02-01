# mondaynightpoker-server

[![.github/workflows/ci.yaml](https://github.com/weters/mondaynightpoker-server/workflows/.github/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/weters/mondaynightpoker-server/actions?query=workflow%3A.github%2Fworkflows%2Fci.yaml+branch%3Amaster)

mondaynightpoker-server is the backend application for the Monday Night Poker site. The front-end code can be found at [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue).

## Supported Card Games

* Bourré
* Pass the Poop
* Poker
  * Seven-card games
    * Follow the Queen
    * Baseball
    * Seven-card Stud
    * Low Card Wild
  * Little L
* Acey Deucey

## Getting Started

### Prerequisites

1. [Go 1.13+](https://golang.org/dl/)
2. [golangci-lint](https://golangci-lint.run/usage/install/)
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
$ RECAPTCHA_SECRET=X go run ./cmd/server
```
    
5. Verify the server is running

```
$ curl http://localhost:5000/health
```
    
6. Start the Vue.js front-end. Repo can be found at [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue)

#### Environment Variables

The following environment variables can be supplied when running the server.

Variable | Default | Description
--- | --- | ---
`PG_DSN` | `postgres://localhost:5000/postgres?sslmode=disable` | PostgreSQL DSN
`MIGRATIONS_PATH` | `./sql` | Path to the database migrations
`JWT_PUBLIC_KEY` | `.keys/public.pem` | Path to the RSA 256 public key for JWT validation
`JWT_PRIVATE_KEY` | `.keys/private.key` | Path to the RSA 256 private key for JWT signing
`RECAPTCHA_SECRET` | | Recaptcha v3 secret key
`DISABLE_ACCESS_LOGS` | | Disables access logging. Only recommended for dev
`START_GAME_DELAY` | `10` | How many seconds to wait after player starts a game
