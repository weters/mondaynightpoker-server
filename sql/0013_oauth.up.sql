BEGIN;

CREATE TABLE oauth_clients
(
    client_id                  text PRIMARY KEY,
    client_secret_hash         text,
    client_name                text      NOT NULL,
    redirect_uris              text[]    NOT NULL,
    grant_types                text[]    NOT NULL,
    token_endpoint_auth_method text      NOT NULL,
    created                    timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE oauth_authorization_codes
(
    code_hash      text PRIMARY KEY,
    client_id      text      NOT NULL REFERENCES oauth_clients (client_id),
    player_id      bigint    NOT NULL REFERENCES players (id),
    redirect_uri   text      NOT NULL,
    code_challenge text      NOT NULL,
    scope          text,
    resource       text,
    expires        timestamp NOT NULL,
    consumed       boolean   NOT NULL DEFAULT FALSE,
    created        timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX oauth_authorization_codes_player_id_idx ON oauth_authorization_codes (player_id);

CREATE TABLE oauth_refresh_tokens
(
    token_hash text PRIMARY KEY,
    client_id  text      NOT NULL REFERENCES oauth_clients (client_id),
    player_id  bigint    NOT NULL REFERENCES players (id),
    scope      text,
    resource   text,
    expires    timestamp NOT NULL,
    revoked    boolean   NOT NULL DEFAULT FALSE,
    rotated_to text,
    created    timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX oauth_refresh_tokens_player_id_idx ON oauth_refresh_tokens (player_id);

COMMIT;
