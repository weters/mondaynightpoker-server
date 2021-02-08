BEGIN;

ALTER TABLE players DROP COLUMN verified;

ALTER TABLE player_tokens DROP COLUMN type;

ALTER TABLE player_tokens RENAME TO player_password_resets;

DROP FUNCTION reset_password(bigint, text, text, timestamp);
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
