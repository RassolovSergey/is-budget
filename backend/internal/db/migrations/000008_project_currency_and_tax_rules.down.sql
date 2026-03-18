DROP TABLE IF EXISTS project_tax_rules;

ALTER TABLE projects
DROP COLUMN IF EXISTS currency_code;
