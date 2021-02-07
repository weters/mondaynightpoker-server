BEGIN;
CREATE TABLE player_password_resets
(
    token     text      not null primary key,
    player_id bigint    not null references players (id),
    active    boolean   not null default 't',
    created   timestamp not null default (now() at time zone 'utc')
);

CREATE INDEX player_password_resets_player_id_idx ON player_password_resets (player_id);

CREATE FUNCTION reset_password(_player_id bigint, _new_password_hash text, _token text, _since timestamp)
    RETURNS bool
    LANGUAGE plpgsql
AS
$$
BEGIN
    PERFORM
    FROM player_password_resets
    WHERE player_id = _player_id
      AND token = _token
      AND active
      AND created > _since
        FOR UPDATE;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    UPDATE players
    SET password_hash = _new_password_hash, updated = (NOW() AT TIME ZONE 'utc')
    WHERE id = _player_id;

    UPDATE player_password_resets
        SET active = 'f'
    WHERE token = _token;

    RETURN true;
END;
$$;
COMMIT;