BEGIN;
DROP FUNCTION reset_password(_player_id bigint, _new_password_hash text, _token text, _since timestamp);
DROP TABLE player_password_resets;
COMMIT;