\if :{?target_email}
\else
\set target_email pm@example.com
\endif

-- Synthetic historical campaigns for forecast tuning.
-- Pattern per campaign:
-- day1 ~40%, day2 still high, middle stagnation, final day ~10%.

WITH target_user AS (
  SELECT COALESCE(
    (SELECT id FROM users WHERE email = :'target_email' LIMIT 1),
    (SELECT id FROM users ORDER BY created_at DESC LIMIT 1)
  ) AS user_id
),
normalized_target AS (
  SELECT user_id
  FROM target_user
  WHERE user_id IS NOT NULL
),
cleanup AS (
  DELETE FROM historical_campaigns hc
  USING normalized_target tu
  WHERE hc.owner_user_id = tu.user_id
    AND hc.project_id IS NULL
    AND hc.name LIKE 'Synthetic %'
  RETURNING hc.id
),
campaign_defs AS (
  SELECT
    gen_random_uuid() AS campaign_id,
    tu.user_id,
    CASE
      WHEN g.idx <= 10 THEN 'Cursed Hamlet'
      ELSE 'Dark Fantasy Village'
    END AS category,
    CASE
      WHEN g.idx <= 10 THEN format('Synthetic Cursed Hamlet %s', lpad(g.idx::text, 2, '0'))
      ELSE format('Synthetic Dark Fantasy Village %s', lpad((g.idx - 10)::text, 2, '0'))
    END AS name,
    (10 + ((g.idx * 7) % 11))::int AS duration_days,
    'EUR'::text AS currency_code,
    (CURRENT_DATE - ((680 - g.idx * 11))::int)::date AS started_at,
    (9000 + g.idx * 950 + ((g.idx % 5) * 700))::numeric(12,2) AS total_income
  FROM generate_series(1, 20) AS g(idx)
  CROSS JOIN normalized_target tu
),
inserted_campaigns AS (
  INSERT INTO historical_campaigns (
    id,
    project_id,
    owner_user_id,
    category,
    name,
    duration_days,
    currency_code,
    started_at,
    ended_at,
    created_at,
    updated_at
  )
  SELECT
    cd.campaign_id,
    NULL,
    cd.user_id,
    cd.category,
    cd.name,
    cd.duration_days,
    cd.currency_code,
    cd.started_at,
    (cd.started_at + (cd.duration_days - 1))::date,
    NOW(),
    NOW()
  FROM campaign_defs cd
  RETURNING id
),
day_plan AS (
  SELECT
    cd.campaign_id,
    d.day_number,
    cd.duration_days,
    cd.total_income,
    CASE
      WHEN d.day_number = 1 THEN 0.40::numeric
      WHEN d.day_number = 2 THEN 0.16::numeric
      WHEN d.day_number = 3 THEN 0.07::numeric
      WHEN d.day_number = cd.duration_days THEN 0.10::numeric
      ELSE (0.27 / (cd.duration_days - 4))::numeric
    END AS day_share
  FROM campaign_defs cd
  JOIN LATERAL generate_series(1, cd.duration_days) AS d(day_number) ON TRUE
),
inserted_metrics AS (
  INSERT INTO historical_campaign_daily_metrics (
    id,
    campaign_id,
    day_number,
    income_amount,
    created_at,
    updated_at
  )
  SELECT
    gen_random_uuid(),
    dp.campaign_id,
    dp.day_number,
    ROUND((dp.total_income * dp.day_share)::numeric, 2)::numeric(12,2),
    NOW(),
    NOW()
  FROM day_plan dp
  RETURNING id, campaign_id, day_number
),
expense_plan AS (
  SELECT
    m.id AS metric_id,
    m.campaign_id,
    m.day_number,
    CASE
      WHEN m.day_number IN (1, 2, 3) THEN 'реклама'
      WHEN m.day_number % 6 = 0 THEN 'производство'
      WHEN m.day_number % 5 = 0 THEN 'логистика'
      ELSE NULL
    END AS category
  FROM inserted_metrics m
),
inserted_expenses AS (
  INSERT INTO historical_campaign_daily_expenses (
    id,
    metric_id,
    category,
    amount,
    created_at,
    updated_at
  )
  SELECT
    gen_random_uuid(),
    ep.metric_id,
    ep.category,
    ROUND(
      CASE
        WHEN ep.category = 'реклама' AND ep.day_number = 1 THEN cd.total_income * 0.050
        WHEN ep.category = 'реклама' AND ep.day_number = 2 THEN cd.total_income * 0.028
        WHEN ep.category = 'реклама' AND ep.day_number = 3 THEN cd.total_income * 0.012
        WHEN ep.category = 'производство' THEN cd.total_income * 0.007
        WHEN ep.category = 'логистика' THEN cd.total_income * 0.0045
        ELSE 0
      END,
      2
    )::numeric(12,2),
    NOW(),
    NOW()
  FROM expense_plan ep
  JOIN campaign_defs cd ON cd.campaign_id = ep.campaign_id
  WHERE ep.category IS NOT NULL
  RETURNING id
)
SELECT
  (SELECT user_id FROM normalized_target) AS seeded_user_id,
  (SELECT count(*) FROM campaign_defs) AS campaigns_seeded,
  (SELECT count(*) FROM inserted_metrics) AS day_rows_seeded,
  (SELECT count(*) FROM inserted_expenses) AS expense_rows_seeded,
  (SELECT COALESCE(sum(total_income), 0) FROM campaign_defs) AS synthetic_total_income;
