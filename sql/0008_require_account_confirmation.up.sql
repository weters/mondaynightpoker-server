BEGIN;

ALTER TABLE player_password_resets
RENAME TO player_tokens;

ALTER TABLE player_tokens
ADD COLUMN type text NOT NULL DEFAULT 'password_reset';

ALTER TABLE player_tokens ALTER COLUMN type DROP DEFAULT;

ALTER TABLE players
ADD COLUMN verified boolean NOT NULL DEFAULT 'f';

UPDATE players SET verified = 't';

DROP FUNCTION reset_password(bigint, text, text, timestamp);
CREATE FUNCTION reset_password(_player_id bigint, _new_password_hash text, _token text, _since timestamp)
    RETURNS bool
    LANGUAGE plpgsql
AS
$$
BEGIN
    PERFORM
    FROM player_tokens
    WHERE player_id = _player_id
      AND token = _token
      AND active
      AND created > _since
      AND type = 'password_reset'
        FOR UPDATE;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    UPDATE players
    SET password_hash = _new_password_hash, updated = (NOW() AT TIME ZONE 'utc')
    WHERE id = _player_id;

    UPDATE player_tokens
    SET active = 'f'
    WHERE token = _token;

    RETURN true;
END;
$$;

COMMIT;