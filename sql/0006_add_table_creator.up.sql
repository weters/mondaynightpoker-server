BEGIN;

ALTER TABLE tables
ADD COLUMN player_id bigint REFERENCES players (id);

CREATE INDEX tables_player_id_created_idx ON tables (player_id, created);

COMMIT;
