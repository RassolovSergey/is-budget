ALTER TABLE projects
ADD COLUMN IF NOT EXISTS secondary_currency_code TEXT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'projects_secondary_currency_code_check'
  ) THEN
    ALTER TABLE projects
    ADD CONSTRAINT projects_secondary_currency_code_check
      CHECK (secondary_currency_code IS NULL OR secondary_currency_code IN ('EUR', 'USD', 'RUB'));
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'projects_secondary_currency_diff_check'
  ) THEN
    ALTER TABLE projects
    ADD CONSTRAINT projects_secondary_currency_diff_check
      CHECK (secondary_currency_code IS NULL OR secondary_currency_code <> currency_code);
  END IF;
END $$;

ALTER TABLE profit_share_rules
ADD COLUMN IF NOT EXISTS currency_code TEXT;

UPDATE profit_share_rules r
SET currency_code = p.currency_code
FROM projects p
WHERE p.id = r.project_id
  AND (r.currency_code IS NULL OR TRIM(r.currency_code) = '');

ALTER TABLE profit_share_rules
ALTER COLUMN currency_code SET NOT NULL;

ALTER TABLE profit_share_rules
ALTER COLUMN currency_code SET DEFAULT 'EUR';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'profit_share_rules_currency_code_check'
  ) THEN
    ALTER TABLE profit_share_rules
    ADD CONSTRAINT profit_share_rules_currency_code_check
      CHECK (currency_code IN ('EUR', 'USD', 'RUB'));
  END IF;
END $$;

ALTER TABLE payouts
ADD COLUMN IF NOT EXISTS currency_code TEXT;

ALTER TABLE payouts
ADD COLUMN IF NOT EXISTS amount_converted NUMERIC(12,2);

ALTER TABLE payouts
ADD COLUMN IF NOT EXISTS conversion_rate NUMERIC(18,6);

UPDATE payouts po
SET currency_code = p.currency_code,
    amount_converted = po.amount,
    conversion_rate = 1
FROM projects p
WHERE p.id = po.project_id
  AND (
    po.currency_code IS NULL
    OR po.amount_converted IS NULL
    OR po.conversion_rate IS NULL
  );

ALTER TABLE payouts
ALTER COLUMN currency_code SET NOT NULL;

ALTER TABLE payouts
ALTER COLUMN amount_converted SET NOT NULL;

ALTER TABLE payouts
ALTER COLUMN conversion_rate SET NOT NULL;

ALTER TABLE payouts
ALTER COLUMN currency_code SET DEFAULT 'EUR';

ALTER TABLE payouts
ALTER COLUMN amount_converted SET DEFAULT 0;

ALTER TABLE payouts
ALTER COLUMN conversion_rate SET DEFAULT 1;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'payouts_currency_code_check'
  ) THEN
    ALTER TABLE payouts
    ADD CONSTRAINT payouts_currency_code_check
      CHECK (currency_code IN ('EUR', 'USD', 'RUB'));
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS currency_rates (
  from_currency_code TEXT NOT NULL,
  to_currency_code TEXT NOT NULL,
  rate NUMERIC(18,6) NOT NULL CHECK (rate > 0),
  source TEXT NOT NULL DEFAULT 'open.er-api.com',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_currency_code, to_currency_code),
  CONSTRAINT currency_rates_from_code_check CHECK (from_currency_code IN ('EUR', 'USD', 'RUB')),
  CONSTRAINT currency_rates_to_code_check CHECK (to_currency_code IN ('EUR', 'USD', 'RUB'))
);

INSERT INTO currency_rates (from_currency_code, to_currency_code, rate, source)
VALUES
  ('EUR', 'EUR', 1, 'identity'),
  ('USD', 'USD', 1, 'identity'),
  ('RUB', 'RUB', 1, 'identity')
ON CONFLICT (from_currency_code, to_currency_code) DO NOTHING;

