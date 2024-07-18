-- Add column index_type to document_collection if it does not exist
DELIMITER $$

CREATE PROCEDURE AddIndexTypeColumnIfNotExists()
BEGIN
    IF NOT EXISTS (SELECT * 
                   FROM INFORMATION_SCHEMA.COLUMNS 
                   WHERE TABLE_NAME = 'document_collection' 
                   AND COLUMN_NAME = 'index_type') THEN
        ALTER TABLE document_collection 
            ADD COLUMN index_type VARCHAR(255);
    END IF;
END$$

CALL AddIndexTypeColumnIfNotExists();
DROP PROCEDURE AddIndexTypeColumnIfNotExists();

DELIMITER ;

-- Update document_collection and set index_type to 'ivfflat' where it is NULL
UPDATE document_collection
    SET index_type = 'ivfflat'
    WHERE index_type IS NULL;
