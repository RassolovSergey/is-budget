INSERT INTO users (id, email, password_hash, full_name)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'pm@example.com', '$2a$10$6xIC1xC.VD.pNzm1jKj3y.KUoabNv5w5RV4CqCIfPz/bE1soMeacm', 'Demo PM'),
  ('11111111-1111-1111-1111-111111111112', 'analyst@example.com', '$2a$10$6xIC1xC.VD.pNzm1jKj3y.KUoabNv5w5RV4CqCIfPz/bE1soMeacm', 'Demo Analyst'),
  ('11111111-1111-1111-1111-111111111113', 'team@example.com', '$2a$10$6xIC1xC.VD.pNzm1jKj3y.KUoabNv5w5RV4CqCIfPz/bE1soMeacm', 'Demo Team'),
  ('11111111-1111-1111-1111-111111111114', 'customer@example.com', '$2a$10$6xIC1xC.VD.pNzm1jKj3y.KUoabNv5w5RV4CqCIfPz/bE1soMeacm', 'Demo Customer');

INSERT INTO projects (id, name, description, created_by)
VALUES
  ('22222222-2222-2222-2222-222222222222', 'Demo Campaign', 'Seeded project for MVP demo', '11111111-1111-1111-1111-111111111111');

INSERT INTO project_members (id, project_id, user_id, role)
VALUES
  ('33333333-3333-3333-3333-333333333331', '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'PM'),
  ('33333333-3333-3333-3333-333333333332', '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111112', 'ANALYST'),
  ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111113', 'TEAM'),
  ('33333333-3333-3333-3333-333333333334', '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111114', 'CUSTOMER');

INSERT INTO planned_expenses (id, project_id, title, category, amount, due_date, is_mandatory, priority, note)
VALUES
  ('44444444-4444-4444-4444-444444444441', '22222222-2222-2222-2222-222222222222', 'Ad Platform Budget', 'ads', 5000.00, CURRENT_DATE + 14, TRUE, 1, 'Mandatory ad spend commitment'),
  ('44444444-4444-4444-4444-444444444442', '22222222-2222-2222-2222-222222222222', 'PR Specialist Fee', 'labor', 1200.00, CURRENT_DATE + 10, TRUE, 2, 'Mandatory specialist payment'),
  ('44444444-4444-4444-4444-444444444443', '22222222-2222-2222-2222-222222222222', 'Concept Art', 'creative', 800.00, CURRENT_DATE + 20, FALSE, 3, 'Optional non-mandatory task');

INSERT INTO actual_income_daily (id, project_id, income_date, amount)
SELECT
  gen_random_uuid(),
  '22222222-2222-2222-2222-222222222222',
  (CURRENT_DATE - INTERVAL '29 day' + (g || ' day')::INTERVAL)::DATE,
  (120 + ((g * 37) % 90))::NUMERIC(12,2)
FROM generate_series(0, 29) AS g;

INSERT INTO actual_expenses (id, project_id, planned_expense_id, title, category, amount, paid_at, note)
VALUES
  ('55555555-5555-5555-5555-555555555551', '22222222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444441', 'Ad Budget - tranche 1', 'ads', 2000.00, CURRENT_DATE - 7, 'First mandatory payment'),
  ('55555555-5555-5555-5555-555555555552', '22222222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444442', 'PR Fee - partial', 'labor', 600.00, CURRENT_DATE - 5, 'Partial mandatory payment'),
  ('55555555-5555-5555-5555-555555555553', '22222222-2222-2222-2222-222222222222', NULL, 'Cloud rendering', 'infra', 350.00, CURRENT_DATE - 3, 'Unplanned operational cost');

INSERT INTO profit_share_rules (id, project_id, recipient_name, percent, applies_after_mandatory)
VALUES
  ('66666666-6666-6666-6666-666666666661', '22222222-2222-2222-2222-222222222222', '3D Artist', 20.00, TRUE);
