CREATE UNIQUE INDEX uq_projects_created_by_name ON projects (created_by, lower(name));
