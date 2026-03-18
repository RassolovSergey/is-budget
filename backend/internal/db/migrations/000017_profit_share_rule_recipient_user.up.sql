ALTER TABLE profit_share_rules
ADD COLUMN IF NOT EXISTS recipient_user_id UUID;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'profit_share_rules_recipient_user_fkey'
  ) THEN
    ALTER TABLE profit_share_rules
    ADD CONSTRAINT profit_share_rules_recipient_user_fkey
      FOREIGN KEY (recipient_user_id) REFERENCES users(id) ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_profit_share_rules_project_recipient_user
ON profit_share_rules(project_id, recipient_user_id)
WHERE recipient_user_id IS NOT NULL;
