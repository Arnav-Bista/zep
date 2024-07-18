-- Check and add column user_id if it does not exist
DELIMITER $$

CREATE PROCEDURE AddUserIdColumnIfNotExists()
BEGIN
    IF NOT EXISTS (SELECT 1 
                   FROM INFORMATION_SCHEMA.COLUMNS 
                   WHERE TABLE_NAME = 'session' 
                   AND COLUMN_NAME = 'user_id') THEN
        ALTER TABLE session ADD COLUMN user_id CHAR(36);
    END IF;
END$$

CALL AddUserIdColumnIfNotExists();
DROP PROCEDURE AddUserIdColumnIfNotExists();

DELIMITER ;

--bun:split

-- Check and add index session_user_id_idx if it does not exist
DELIMITER $$

CREATE PROCEDURE AddUserIdIndexIfNotExists()
BEGIN
    IF NOT EXISTS (SELECT 1 
                   FROM INFORMATION_SCHEMA.STATISTICS 
                   WHERE TABLE_NAME = 'session' 
                   AND INDEX_NAME = 'session_user_id_idx') THEN
        CREATE INDEX session_user_id_idx ON session(user_id);
    END IF;
END$$

CALL AddUserIdIndexIfNotExists();
DROP PROCEDURE AddUserIdIndexIfNotExists();

DELIMITER ;

--bun:split

-- Check and add column id if it does not exist
DELIMITER $$

CREATE PROCEDURE AddIdColumnIfNotExists()
BEGIN
    IF NOT EXISTS (SELECT 1 
                   FROM INFORMATION_SCHEMA.COLUMNS 
                   WHERE TABLE_NAME = 'session' 
                   AND COLUMN_NAME = 'id') THEN
        ALTER TABLE session ADD COLUMN id BIGINT AUTO_INCREMENT PRIMARY KEY;
    END IF;
END$$

CALL AddIdColumnIfNotExists();
DROP PROCEDURE AddIdColumnIfNotExists();

DELIMITER ;
