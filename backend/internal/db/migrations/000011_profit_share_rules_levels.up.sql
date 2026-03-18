ALTER TABLE profit_share_rules
ADD COLUMN level INTEGER;

UPDATE profit_share_rules
SET level = CASE
  WHEN applies_after_mandatory THEN 2
  ELSE 1
END;

ALTER TABLE profit_share_rules
ALTER COLUMN level SET NOT NULL,
ALTER COLUMN level SET DEFAULT 1;

ALTER TABLE profit_share_rules
ADD CONSTRAINT profit_share_rules_level_check CHECK (level >= 1);

CREATE INDEX IF NOT EXISTS idx_profit_share_rules_project_level_created
ON profit_share_rules(project_id, level, created_at);
