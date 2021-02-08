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
$ MNP_RECAPTCHA_SECRET=X go run ./cmd/server
```
    
5. Verify the server is running

```
$ curl http://localhost:5000/health
```
    
6. Start the Vue.js front-end. Repo can be found at [github.com/weters/mondaynightpoker-vue](https://github.com/weters/mondaynightpoker-vue)

### Configuration

The service can be configured through two methods:

1. **YAML Configuration:** By default, the service will look for `config.yaml`. You can also change the filename by setting a `MNP_CONFIG_FILE` environment variable.
2. **Environment Variables:** All configuration settings can be set by environment variables. Every variable is prefixed by `MNP_` and `camelCase` is transformed to `SNAKE_CASE`. Example, `jwt.publicKey` will become `MNP_JWT_PUBLIC_KEY`.

Any environment variables take precedence over values defined in YAML. The default configuration values are defined below.

```yaml
host: https://monday-night.poker
logLevel: info
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
  from: Monday Night Poker <no-replay@monday-night.poker>
  sender: no-reply@monday-night.poker
  username: dealer@monday-night.poker
  password: ""
  host: mail.privateemail.com:587
  templatesDir: templates
  disable: false
```

You can generate a YAML file with the defaults by running:

```shell
$ go run ./cmd/generate-config
```