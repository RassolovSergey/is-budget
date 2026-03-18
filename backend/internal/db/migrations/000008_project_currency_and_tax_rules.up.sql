ALTER TABLE projects
ADD COLUMN currency_code TEXT NOT NULL DEFAULT 'EUR'
CHECK (currency_code IN ('EUR', 'USD', 'RUB'));

CREATE TABLE project_tax_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  percent NUMERIC(5,2) NOT NULL CHECK (percent >= 0 AND percent <= 100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_tax_rules_project_id ON project_tax_rules(project_id);

INSERT INTO project_tax_rules (id, project_id, name, percent)
SELECT gen_random_uuid(), p.id, 'Комиссия платформы', 12.00
FROM projects p
WHERE p.id = '22222222-2222-2222-2222-222222222222';
