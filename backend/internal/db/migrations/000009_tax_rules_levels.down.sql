DROP INDEX IF EXISTS idx_project_tax_rules_project_level;

ALTER TABLE project_tax_rules
DROP CONSTRAINT IF EXISTS project_tax_rules_level_check;

ALTER TABLE project_tax_rules
DROP COLUMN IF EXISTS level;
