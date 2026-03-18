ALTER TABLE projects
ADD COLUMN duration_days INTEGER NOT NULL DEFAULT 14,
ADD COLUMN status TEXT NOT NULL DEFAULT 'preparation';

ALTER TABLE projects
ADD CONSTRAINT projects_duration_days_check CHECK (duration_days BETWEEN 3 AND 30),
ADD CONSTRAINT projects_status_check CHECK (status IN ('preparation', 'active', 'completed'));
