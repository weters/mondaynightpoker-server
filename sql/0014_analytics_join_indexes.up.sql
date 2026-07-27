BEGIN;

-- Postgres does not index foreign keys automatically, so both of these join
-- columns were unindexed. Every read that resolves a game to its ledger rows
-- (get_game, list_table_games) scanned the whole transactions table, and every
-- read scoped to one table scanned every game site-wide.

CREATE INDEX players_tables_transactions_game_id_idx ON players_tables_transactions(game_id);

CREATE INDEX games_table_uuid_idx ON games(table_uuid);

COMMIT;
