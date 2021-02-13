BEGIN;

ALTER TABLE players ADD COLUMN verified boolean not null default 'f';

UPDATE players SET verified = 't' WHERE status = 'verified';

ALTER TABLE players DROP COLUMN status;
DROP TYPE player_status;

COMMIT;