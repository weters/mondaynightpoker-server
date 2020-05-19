BEGIN;
ALTER TABLE players_tables
    ADD COLUMN is_blocked boolean not null default false;
END;