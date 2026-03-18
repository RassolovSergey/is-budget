ALTER TABLE project_tax_rules
ADD COLUMN level INTEGER;

WITH ranked AS (
  SELECT id,
         ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY created_at, id) AS rn
  FROM project_tax_rules
)
UPDATE project_tax_rules ptr
SET level = ranked.rn
FROM ranked
WHERE ptr.id = ranked.id;

ALTER TABLE project_tax_rules
ALTER COLUMN level SET NOT NULL,
ALTER COLUMN level SET DEFAULT 1;

ALTER TABLE project_tax_rules
ADD CONSTRAINT project_tax_rules_level_check CHECK (level >= 1);

CREATE UNIQUE INDEX idx_project_tax_rules_project_level
ON project_tax_rules(project_id, level);
