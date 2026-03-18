WITH duplicates AS (
  SELECT
    id,
    name,
    ROW_NUMBER() OVER (PARTITION BY created_by, lower(name) ORDER BY created_at, id) AS rn
  FROM projects
),
normalized AS (
  SELECT
    id,
    CASE
      WHEN rn = 1 THEN name
      ELSE CONCAT(name, ' #', SUBSTRING(id::text, 1, 8))
    END AS normalized_name
  FROM duplicates
)
UPDATE projects p
SET
  name = n.normalized_name,
  updated_at = NOW()
FROM normalized n
WHERE p.id = n.id
  AND p.name <> n.normalized_name;

CREATE UNIQUE INDEX uq_projects_created_by_name ON projects (created_by, lower(name));
