DROP INDEX IF EXISTS idx_project_tax_rules_project_level;

CREATE INDEX IF NOT EXISTS idx_project_tax_rules_project_level_created
ON project_tax_rules(project_id, level, created_at);
