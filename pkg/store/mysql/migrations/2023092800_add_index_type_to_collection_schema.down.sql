-- Drop column index_type if it exists
DELIMITER $$

CREATE PROCEDURE DropIndexTypeColumnIfExists()
BEGIN
    IF EXISTS (SELECT * 
               FROM INFORMATION_SCHEMA.COLUMNS 
               WHERE TABLE_NAME = 'document_collection' 
               AND COLUMN_NAME = 'index_type') THEN
        ALTER TABLE document_collection 
            DROP COLUMN index_type;
    END IF;
END$$

CALL DropIndexTypeColumnIfExists();
DROP PROCEDURE DropIndexTypeColumnIfExists();

DELIMITER ;
