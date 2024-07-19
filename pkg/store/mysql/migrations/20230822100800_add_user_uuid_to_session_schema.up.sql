DELIMITER $$

-- Add user_id column if not exists
CREATE PROCEDURE AddUserIdColumnIfNotExists()
BEGIN
    IF NOT EXISTS(
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'session'
          AND column_name = 'user_id') THEN
        ALTER TABLE session
            ADD COLUMN user_id CHAR(36);
    END IF;
END$$

CALL AddUserIdColumnIfNotExists();
DROP PROCEDURE AddUserIdColumnIfNotExists();

DELIMITER ; -- Reset the delimiter

--bun:split

DELIMITER $$

-- Create index session_user_id_idx if not exists
CREATE PROCEDURE AddSessionUserIdIndexIfNotExists()
BEGIN
    IF NOT EXISTS(
        SELECT 1
        FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'session'
          AND index_name = 'session_user_id_idx') THEN
        CREATE INDEX session_user_id_idx ON session(user_id);
    END IF;
END$$

CALL AddSessionUserIdIndexIfNotExists();
DROP PROCEDURE AddSessionUserIdIndexIfNotExists();

DELIMITER ; -- Reset the delimiter

--bun:split

DELIMITER $$

-- Add id column if not exists
CREATE PROCEDURE AddIdColumnIfNotExists()
BEGIN
    IF NOT EXISTS(
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'session'
          AND column_name = 'id') THEN
        ALTER TABLE session
            ADD COLUMN id BIGINT AUTO_INCREMENT PRIMARY KEY;
    END IF;
END$$

CALL AddIdColumnIfNotExists();
DROP PROCEDURE AddIdColumnIfNotExists();

DELIMITER ; -- Reset the delimiter
