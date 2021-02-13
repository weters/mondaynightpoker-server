BEGIN;

CREATE TYPE player_status AS enum ('created', 'verified', 'blocked', 'deleted');
ALTER TABLE players
ADD COLUMN status player_status NOT NULL default 'created';

UPDATE players SET status = 'verified' WHERE verified = 't';

ALTER TABLE players DROP COLUMN verified;

CREATE INDEX players_status_idx ON players(status);

COMMIT;
