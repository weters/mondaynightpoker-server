BEGIN;

DROP INDEX players_lower_email_idx;
ALTER TABLE players ADD CONSTRAINT players_email_key UNIQUE (email);

COMMIT;