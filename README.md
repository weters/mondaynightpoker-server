# mondaynightpoker-server

![.github/workflows/ci.yaml](https://github.com/weters/mondaynightpoker-server/workflows/.github/workflows/ci.yaml/badge.svg)

## Environment Variables

Variable | Default | Description
--- | --- | ---
`PG_DSN` | `postgres://localhost:5000/postgres?sslmode=disable` | PostgreSQL DSN
`MIGRATIONS_PATH` | `./sql` | Path to the database migrations
`JWT_PUBLIC_KEY` | `.keys/public.pem` | Path to the RSA 256 public key for JWT validation
`JWT_PRIVATE_KEY` | `.keys/private.key` | Path to the RSA 256 private key for JWT signing
`RECAPTCHA_SECRET` | | Recaptcha v3 secret key
