ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_default_currency_code_check;

ALTER TABLE users
DROP COLUMN IF EXISTS default_currency_code;
