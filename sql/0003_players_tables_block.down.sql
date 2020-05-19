BEGIN;
ALTER TABLE players_tables
    DROP COLUMN is_blocked;
END;