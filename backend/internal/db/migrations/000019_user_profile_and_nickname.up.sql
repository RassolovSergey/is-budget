ALTER TABLE users
ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS nickname TEXT,
ADD COLUMN IF NOT EXISTS avatar_path TEXT;

UPDATE users
SET first_name = split_part(trim(full_name), ' ', 1),
    last_name = trim(regexp_replace(trim(full_name), '^[^ ]+\s*', ''))
WHERE first_name = '' AND last_name = '';

WITH base AS (
  SELECT id,
         COALESCE(NULLIF(lower(trim(regexp_replace(split_part(email, '@', 1), '[^a-zA-Z0-9_]+', '_', 'g'))), ''), 'user') AS base_name
  FROM users
),
normalized AS (
  SELECT id,
         CASE
           WHEN length(base_name) < 3 THEN 'user_' || substr(replace(id::text, '-', ''), 1, 8)
           ELSE base_name
         END AS normalized_name
  FROM base
),
ranked AS (
  SELECT id,
         normalized_name,
         row_number() OVER (PARTITION BY normalized_name ORDER BY id) AS rn
  FROM normalized
)
UPDATE users u
SET nickname = CASE
                 WHEN r.rn = 1 THEN left(r.normalized_name, 32)
                 ELSE left(r.normalized_name, 28) || '_' || r.rn::text
               END
FROM ranked r
WHERE u.id = r.id
  AND (u.nickname IS NULL OR trim(u.nickname) = '');

ALTER TABLE users
ALTER COLUMN nickname SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'users_nickname_format_check'
  ) THEN
    ALTER TABLE users
    ADD CONSTRAINT users_nickname_format_check
      CHECK (nickname ~ '^[a-z0-9_]{3,32}$');
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_nickname_unique
ON users (nickname);
