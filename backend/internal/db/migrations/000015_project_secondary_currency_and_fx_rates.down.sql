DROP TABLE IF EXISTS currency_rates;

ALTER TABLE payouts
DROP CONSTRAINT IF EXISTS payouts_currency_code_check;

ALTER TABLE payouts
DROP COLUMN IF EXISTS conversion_rate;

ALTER TABLE payouts
DROP COLUMN IF EXISTS amount_converted;

ALTER TABLE payouts
DROP COLUMN IF EXISTS currency_code;

ALTER TABLE profit_share_rules
DROP CONSTRAINT IF EXISTS profit_share_rules_currency_code_check;

ALTER TABLE profit_share_rules
DROP COLUMN IF EXISTS currency_code;

ALTER TABLE projects
DROP CONSTRAINT IF EXISTS projects_secondary_currency_diff_check;

ALTER TABLE projects
DROP CONSTRAINT IF EXISTS projects_secondary_currency_code_check;

ALTER TABLE projects
DROP COLUMN IF EXISTS secondary_currency_code;

