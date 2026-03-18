-- name: ListPlannedExpensesByProject :many
SELECT id,
       project_id,
       title,
       category,
       amount::double precision AS amount,
       due_date,
       is_mandatory,
       priority,
       note,
       created_at,
       updated_at
FROM planned_expenses
WHERE project_id = $1
ORDER BY priority ASC, due_date ASC;

-- name: GetPlannedExpenseByID :one
SELECT id,
       project_id,
       title,
       category,
       amount::double precision AS amount,
       due_date,
       is_mandatory,
       priority,
       note,
       created_at,
       updated_at
FROM planned_expenses
WHERE project_id = $1 AND id = $2;

-- name: CreatePlannedExpense :one
INSERT INTO planned_expenses (
  id, project_id, title, category, amount, due_date, is_mandatory, priority, note
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id,
          project_id,
          title,
          category,
          amount::double precision AS amount,
          due_date,
          is_mandatory,
          priority,
          note,
          created_at,
          updated_at;

-- name: UpdatePlannedExpense :one
UPDATE planned_expenses
SET title = $3,
    category = $4,
    amount = $5,
    due_date = $6,
    is_mandatory = $7,
    priority = $8,
    note = $9,
    updated_at = NOW()
WHERE project_id = $1 AND id = $2
RETURNING id,
          project_id,
          title,
          category,
          amount::double precision AS amount,
          due_date,
          is_mandatory,
          priority,
          note,
          created_at,
          updated_at;

-- name: DeletePlannedExpense :exec
DELETE FROM planned_expenses
WHERE project_id = $1 AND id = $2;
