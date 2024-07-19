DELIMITER $$

-- Add column index_type to document_collection if it does not exist
CREATE PROCEDURE AddIndexTypeColumnIfNotExists()
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'document_collection'
          AND column_name = 'index_type') THEN
        ALTER TABLE document_collection
            ADD COLUMN index_type TEXT;
    END IF;
END$$

CALL AddIndexTypeColumnIfNotExists();
DROP PROCEDURE AddIndexTypeColumnIfNotExists();

DELIMITER ;

-- Update document_collection and set index_type to 'ivfflat' where it is NULL
UPDATE document_collection
SET index_type = 'ivfflat'
WHERE index_type IS NULL;
