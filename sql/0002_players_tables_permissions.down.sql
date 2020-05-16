BEGIN;
ALTER TABLE players_tables
DROP COLUMN can_restart,
    DROP COLUMN can_terminate,
    DROP COLUMN can_start;
END;