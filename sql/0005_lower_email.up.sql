BEGIN;

ALTER TABLE players DROP CONSTRAINT players_email_key;
CREATE UNIQUE INDEX players_lower_email_idx ON players (LOWER(email));

COMMIT;