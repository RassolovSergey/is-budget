DROP INDEX IF EXISTS idx_users_nickname_unique;

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_nickname_format_check;

ALTER TABLE users
DROP COLUMN IF EXISTS avatar_path,
DROP COLUMN IF EXISTS nickname,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS first_name;
