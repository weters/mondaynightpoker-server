BEGIN;

CREATE TABLE tables
(
    uuid    uuid PRIMARY KEY,
    name    text,
    created timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE players
(
    id            bigserial PRIMARY KEY,
    email         text UNIQUE,
    display_name  text,
    is_site_admin boolean   NOT NULL DEFAULT FALSE,
    password_hash text      NOT NULL,
    remote_addr   text      NOT NULL,
    created       timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated       timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX players_remote_addr_idx ON players (remote_addr);

CREATE TABLE players_tables
(
    id             bigserial PRIMARY KEY,
    player_id      bigint    NOT NULL REFERENCES players (id),
    table_uuid     uuid      NOT NULL REFERENCES tables (uuid),
    is_table_admin boolean   NOT NULL DEFAULT FALSE,
    balance        int       NOT NULL DEFAULT 0,
    active         boolean   NOT NULL DEFAULT TRUE,
    created        timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated        timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    UNIQUE (player_id, table_uuid)
);

CREATE INDEX players_tables_table_uuid_idx ON players_tables (table_uuid);

CREATE TYPE game_t AS enum ('bourre');

CREATE TABLE games
(
    id         bigserial PRIMARY KEY,
    parent_id  bigint REFERENCES games (id),
    table_uuid uuid      NOT NULL REFERENCES tables (uuid),
    game_type  game_t    NOT NULL,
    data       jsonb,
    created    timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    ended      timestamp
);

CREATE TABLE players_tables_transactions
(
    id                bigserial PRIMARY KEY,
    players_tables_id bigint    NOT NULL REFERENCES players_tables (id),
    adjustment        int       NOT NULL,
    previous_balance  int       NOT NULL,
    current_balance   int       NOT NULL,
    game_id           bigint references games (id),
    reason            text,
    created           timestamp NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE FUNCTION adjust_balance(_players_tables_id bigint, _current_balance int, _adjustment int,
                               _game_id bigint, _reason text) RETURNS boolean
    LANGUAGE plpgsql
AS
$$
DECLARE
    _previous_balance int;
    _next_balance     int;
BEGIN
    SELECT INTO _previous_balance balance FROM players_tables WHERE id = _players_tables_id FOR UPDATE;
    IF NOT found THEN
        RAISE EXCEPTION 'could not find current amount for id %', _players_tables_id;
    END IF;

    IF _previous_balance != _current_balance THEN
        RAISE EXCEPTION 'balance has changed';
    END IF;

    _next_balance := _previous_balance + _adjustment;

    INSERT INTO players_tables_transactions (players_tables_id, adjustment, previous_balance,
                                             current_balance, game_id, reason)
    VALUES (_players_tables_id, _adjustment, _previous_balance, _next_balance, _game_id, _reason);

    UPDATE players_tables SET balance = _next_balance, updated = NOW() WHERE id = _players_tables_id;

    RETURN TRUE;
END;
$$;

COMMIT;
