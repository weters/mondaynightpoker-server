BEGIN;

ALTER TABLE tables
    ADD COLUMN modified TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT 'f';
CREATE INDEX tables_deleted_idx ON tables (deleted);

COMMIT;