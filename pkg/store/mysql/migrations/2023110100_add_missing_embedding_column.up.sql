-- Check and add column embedding to message_embedding if it exists
DELIMITER $$

CREATE PROCEDURE AddEmbeddingColumnToMessageEmbeddingIfExists()
BEGIN
    IF (SELECT COUNT(*) 
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_NAME = 'message_embedding') > 0 THEN
        IF NOT EXISTS (SELECT * 
                       FROM INFORMATION_SCHEMA.COLUMNS 
                       WHERE TABLE_NAME = 'message_embedding' 
                       AND COLUMN_NAME = 'embedding') THEN
            ALTER TABLE message_embedding 
                ADD COLUMN embedding JSON;
        END IF;
    END IF;
END$$

CALL AddEmbeddingColumnToMessageEmbeddingIfExists();
DROP PROCEDURE AddEmbeddingColumnToMessageEmbeddingIfExists();

DELIMITER ;

--bun:split

-- Check and add column embedding to summary_embedding if it exists
DELIMITER $$

CREATE PROCEDURE AddEmbeddingColumnToSummaryEmbeddingIfExists()
BEGIN
    IF (SELECT COUNT(*) 
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_NAME = 'summary_embedding') > 0 THEN
        IF NOT EXISTS (SELECT * 
                       FROM INFORMATION_SCHEMA.COLUMNS 
                       WHERE TABLE_NAME = 'summary_embedding' 
                       AND COLUMN_NAME = 'embedding') THEN
            ALTER TABLE summary_embedding 
                ADD COLUMN embedding JSON;
        END IF;
    END IF;
END$$

CALL AddEmbeddingColumnToSummaryEmbeddingIfExists();
DROP PROCEDURE AddEmbeddingColumnToSummaryEmbeddingIfExists();

DELIMITER ;
