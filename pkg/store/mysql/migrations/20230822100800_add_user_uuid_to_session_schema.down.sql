-- Drop index if it exists
DROP INDEX IF EXISTS session_user_id_idx ON session;

--bun:split

-- Drop user_id column if it exists
ALTER TABLE session
    DROP COLUMN IF EXISTS user_id;

--bun:split

-- Drop id column if it exists
ALTER TABLE session
    DROP COLUMN IF EXISTS id;
