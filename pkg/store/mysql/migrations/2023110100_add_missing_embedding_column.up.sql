DELIMITER $$

-- Add column embedding to message_embedding if it exists and column does not exist
CREATE PROCEDURE AddEmbeddingColumnToMessageEmbeddingIfExists()
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = DATABASE()
          AND table_name = 'message_embedding') THEN
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'message_embedding'
              AND column_name = 'embedding') THEN
            ALTER TABLE message_embedding
                ADD COLUMN embedding JSON; -- Using JSON as an alternative to vector type
        END IF;
    END IF;
END$$

CALL AddEmbeddingColumnToMessageEmbeddingIfExists();
DROP PROCEDURE AddEmbeddingColumnToMessageEmbeddingIfExists();

DELIMITER ;

--bun:split

DELIMITER $$

-- Add column embedding to summary_embedding if it exists and column does not exist
CREATE PROCEDURE AddEmbeddingColumnToSummaryEmbeddingIfExists()
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = DATABASE()
          AND table_name = 'summary_embedding') THEN
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'summary_embedding'
              AND column_name = 'embedding') THEN
            ALTER TABLE summary_embedding
                ADD COLUMN embedding JSON; -- Using JSON as an alternative to vector type
        END IF;
    END IF;
END$$

CALL AddEmbeddingColumnToSummaryEmbeddingIfExists();
DROP PROCEDURE AddEmbeddingColumnToSummaryEmbeddingIfExists();

DELIMITER ;
