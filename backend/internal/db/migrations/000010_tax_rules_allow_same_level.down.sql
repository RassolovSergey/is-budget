DROP INDEX IF EXISTS idx_project_tax_rules_project_level_created;

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_tax_rules_project_level
ON project_tax_rules(project_id, level);
