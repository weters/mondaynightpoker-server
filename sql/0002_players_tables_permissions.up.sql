BEGIN;
ALTER TABLE players_tables
    ADD COLUMN can_start boolean not null default false,
    ADD COLUMN can_restart boolean not null default false,
    ADD COLUMN can_terminate boolean not null default false;
END;