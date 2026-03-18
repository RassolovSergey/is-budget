-- name: GetFinanceSummaryByProject :one
SELECT
  COALESCE((SELECT SUM(aid.amount) FROM actual_income_daily aid WHERE aid.project_id = $1), 0)::double precision AS income_total,
  COALESCE((SELECT SUM(ae.amount) FROM actual_expenses ae WHERE ae.project_id = $1), 0)::double precision AS expenses_total,
  COALESCE((SELECT SUM(pe.amount) FROM planned_expenses pe WHERE pe.project_id = $1), 0)::double precision AS planned_total,
  COALESCE((SELECT SUM(pe.amount) FROM planned_expenses pe WHERE pe.project_id = $1 AND pe.is_mandatory = TRUE), 0)::double precision AS mandatory_total,
  COALESCE((
    SELECT SUM(ae.amount)
    FROM actual_expenses ae
    JOIN planned_expenses pe ON pe.id = ae.planned_expense_id
    WHERE ae.project_id = $1 AND pe.is_mandatory = TRUE
  ), 0)::double precision AS mandatory_paid;
