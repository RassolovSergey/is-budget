DROP INDEX IF EXISTS idx_profit_share_rules_project_level_created;

ALTER TABLE profit_share_rules
DROP CONSTRAINT IF EXISTS profit_share_rules_level_check;

ALTER TABLE profit_share_rules
DROP COLUMN IF EXISTS level;
