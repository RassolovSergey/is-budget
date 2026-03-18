DO $$
DECLARE
  c RECORD;
BEGIN
  FOR c IN
    SELECT conname
    FROM pg_constraint
    WHERE conrelid = 'historical_campaign_daily_metrics'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) ILIKE '%income_amount%'
      AND pg_get_constraintdef(oid) ILIKE '%>= 0%'
  LOOP
    EXECUTE format('ALTER TABLE historical_campaign_daily_metrics DROP CONSTRAINT IF EXISTS %I', c.conname);
  END LOOP;
END$$;

