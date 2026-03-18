ALTER TABLE actual_income_daily
  DROP CONSTRAINT IF EXISTS actual_income_daily_amount_check;

-- Normalize possible negative values before restoring non-negative constraint.
UPDATE actual_income_daily
SET amount = 0
WHERE amount < 0;

ALTER TABLE actual_income_daily
  ADD CONSTRAINT actual_income_daily_amount_check CHECK (amount >= 0);
