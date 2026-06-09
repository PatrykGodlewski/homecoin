-- Dev seed: one household with 3 members.
-- Password for all accounts: password123
-- Idempotent: safe to run multiple times.

BEGIN;

INSERT INTO users (email, password_hash, display_name, created_at, updated_at)
VALUES
    ('alice@homecoin.test', '$2b$10$mla/Hfv1NM3DscVp537geeAshCJspZurDkDJzawnrXfbbUURunAhC', 'Alice', NOW(), NOW()),
    ('bob@homecoin.test',   '$2b$10$mla/Hfv1NM3DscVp537geeAshCJspZurDkDJzawnrXfbbUURunAhC', 'Bob',   NOW(), NOW()),
    ('carol@homecoin.test', '$2b$10$mla/Hfv1NM3DscVp537geeAshCJspZurDkDJzawnrXfbbUURunAhC', 'Carol', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

INSERT INTO households (name, currency, invite_code, created_at, updated_at)
SELECT 'The Apartment', 'USD', 'demo1234', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM households WHERE invite_code = 'demo1234');

INSERT INTO household_members (household_id, user_id, role, joined_at)
SELECT h.id, u.id, 'owner', NOW()
FROM households h
JOIN users u ON u.email = 'alice@homecoin.test'
WHERE h.invite_code = 'demo1234'
  AND NOT EXISTS (SELECT 1 FROM household_members hm WHERE hm.user_id = u.id)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO household_members (household_id, user_id, role, joined_at)
SELECT h.id, u.id, 'member', NOW()
FROM households h
JOIN users u ON u.email = 'bob@homecoin.test'
WHERE h.invite_code = 'demo1234'
  AND NOT EXISTS (SELECT 1 FROM household_members hm WHERE hm.user_id = u.id)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO household_members (household_id, user_id, role, joined_at)
SELECT h.id, u.id, 'member', NOW()
FROM households h
JOIN users u ON u.email = 'carol@homecoin.test'
WHERE h.invite_code = 'demo1234'
  AND NOT EXISTS (SELECT 1 FROM household_members hm WHERE hm.user_id = u.id)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO categories (household_id, name, icon, color, is_fixed, created_at, updated_at)
SELECT h.id, v.name, v.icon, v.color, v.is_fixed, NOW(), NOW()
FROM households h
CROSS JOIN (VALUES
    ('Rent',          'home',        '#6366F1', TRUE),
    ('Groceries',     'cart',        '#22C55E', FALSE),
    ('Utilities',     'bolt',        '#F59E0B', TRUE),
    ('Transport',     'car',         '#3B82F6', FALSE),
    ('Entertainment', 'film',        '#EC4899', FALSE),
    ('Dining Out',    'utensils',    '#EF4444', FALSE),
    ('Healthcare',    'heart',       '#14B8A6', FALSE),
    ('Savings',       'piggy-bank',  '#8B5CF6', FALSE)
) AS v(name, icon, color, is_fixed)
WHERE h.invite_code = 'demo1234'
ON CONFLICT (household_id, name) DO NOTHING;

COMMIT;
