ALTER TABLE actual_income_daily
  DROP CONSTRAINT IF EXISTS actual_income_daily_amount_check;

-- Keep NUMERIC(12,2), allow negative daily deltas (refunds/chargebacks).
ALTER TABLE actual_income_daily
  ADD CONSTRAINT actual_income_daily_amount_check CHECK (amount >= -9999999999.99 AND amount <= 9999999999.99);
