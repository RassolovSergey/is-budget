-- name: ListActualExpensesByProjectRange :many
SELECT id,
       project_id,
       planned_expense_id,
       title,
       category,
       amount::double precision AS amount,
       paid_at,
       note,
       created_at,
       updated_at
FROM actual_expenses
WHERE project_id = $1
  AND paid_at BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date)
ORDER BY paid_at DESC, created_at DESC;

-- name: ListActualExpensesByProject :many
SELECT id,
       project_id,
       planned_expense_id,
       title,
       category,
       amount::double precision AS amount,
       paid_at,
       note,
       created_at,
       updated_at
FROM actual_expenses
WHERE project_id = $1
ORDER BY paid_at DESC, created_at DESC;

-- name: CreateActualExpense :one
INSERT INTO actual_expenses (
  id, project_id, planned_expense_id, title, category, amount, paid_at, note
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
RETURNING id,
          project_id,
          planned_expense_id,
          title,
          category,
          amount::double precision AS amount,
          paid_at,
          note,
          created_at,
          updated_at;

-- name: UpdateActualExpense :one
UPDATE actual_expenses
SET planned_expense_id = $3,
    title = $4,
    category = $5,
    amount = $6,
    paid_at = $7,
    note = $8,
    updated_at = NOW()
WHERE project_id = $1 AND id = $2
RETURNING id,
          project_id,
          planned_expense_id,
          title,
          category,
          amount::double precision AS amount,
          paid_at,
          note,
          created_at,
          updated_at;

-- name: DeleteActualExpense :exec
DELETE FROM actual_expenses
WHERE project_id = $1 AND id = $2;
