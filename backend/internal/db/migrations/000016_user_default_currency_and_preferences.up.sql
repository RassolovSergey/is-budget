ALTER TABLE users
ADD COLUMN IF NOT EXISTS default_currency_code TEXT NOT NULL DEFAULT 'EUR';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'users_default_currency_code_check'
  ) THEN
    ALTER TABLE users
    ADD CONSTRAINT users_default_currency_code_check
      CHECK (default_currency_code IN ('EUR', 'USD', 'RUB'));
  END IF;
END $$;
