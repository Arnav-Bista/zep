DELIMITER $$

-- Drop the index_type column if it exists
CREATE PROCEDURE DropIndexTypeColumnIfExists()
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'document_collection'
          AND column_name = 'index_type') THEN
        ALTER TABLE document_collection
            DROP COLUMN index_type;
    END IF;
END$$

CALL DropIndexTypeColumnIfExists();
DROP PROCEDURE DropIndexTypeColumnIfExists();

DELIMITER ;
