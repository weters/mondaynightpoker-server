BEGIN;

-- The up migration adds both of these columns; dropping only `deleted` left
-- `modified` behind, so re-applying the migration failed on the column already
-- existing and left the schema_migrations row dirty. Dropping `deleted` also
-- drops tables_deleted_idx, so the index needs no separate statement.
ALTER TABLE tables
    DROP COLUMN deleted,
    DROP COLUMN modified;

COMMIT;