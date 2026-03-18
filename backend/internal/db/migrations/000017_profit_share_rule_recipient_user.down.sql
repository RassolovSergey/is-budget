ALTER TABLE profit_share_rules
DROP CONSTRAINT IF EXISTS profit_share_rules_recipient_user_fkey;

DROP INDEX IF EXISTS idx_profit_share_rules_project_recipient_user;

ALTER TABLE profit_share_rules
DROP COLUMN IF EXISTS recipient_user_id;
