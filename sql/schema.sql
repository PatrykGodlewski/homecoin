-- HomeCoin initial schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE split_type AS ENUM ('equal', 'exact', 'percentage', 'shares');
CREATE TYPE settlement_status AS ENUM ('pending', 'confirmed', 'rejected');
CREATE TYPE reminder_status AS ENUM ('scheduled', 'sent', 'dismissed');
CREATE TYPE budget_period AS ENUM ('weekly', 'monthly', 'yearly');
CREATE TYPE alert_status AS ENUM ('active', 'acknowledged', 'resolved');
CREATE TYPE piggy_bank_status AS ENUM ('active', 'completed', 'archived');
CREATE TYPE notification_channel AS ENUM ('in_app', 'email', 'push');
CREATE TYPE notification_type AS ENUM (
    'expense_added', 'debt_reminder', 'budget_threshold',
    'settlement_request', 'piggy_bank_milestone', 'household_invite'
);

-- Users
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    monthly_income_cents BIGINT,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Households
CREATE TABLE households (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    currency        CHAR(3) NOT NULL DEFAULT 'USD',
    invite_code     VARCHAR(32) UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Household members (one household per user)
CREATE TABLE household_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            user_role NOT NULL DEFAULT 'member',
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, user_id),
    UNIQUE (user_id)
);

CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Categories
CREATE TABLE categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    icon            VARCHAR(50),
    color           CHAR(7),
    is_fixed        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, name)
);

-- Expenses
CREATE TABLE expenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    payer_id        UUID NOT NULL REFERENCES users(id),
    category_id     UUID REFERENCES categories(id),
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    split_type      split_type NOT NULL DEFAULT 'equal',
    expense_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE expense_splits (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id          UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    debtor_id           UUID NOT NULL REFERENCES users(id),
    amount_cents        BIGINT NOT NULL CHECK (amount_cents >= 0),
    exact_amount_cents  BIGINT,
    percentage          NUMERIC(5,2),
    shares              NUMERIC(10,2),
    UNIQUE (expense_id, debtor_id)
);

-- Balances
CREATE TABLE household_balances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    creditor_id     UUID NOT NULL REFERENCES users(id),
    debtor_id       UUID NOT NULL REFERENCES users(id),
    balance_cents   BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, creditor_id, debtor_id),
    CHECK (creditor_id <> debtor_id)
);

CREATE TABLE settlements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    from_user_id    UUID NOT NULL REFERENCES users(id),
    to_user_id      UUID NOT NULL REFERENCES users(id),
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    status          settlement_status NOT NULL DEFAULT 'pending',
    note            TEXT,
    settled_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE debt_reminders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    creditor_id     UUID NOT NULL REFERENCES users(id),
    debtor_id       UUID NOT NULL REFERENCES users(id),
    amount_cents    BIGINT NOT NULL,
    status          reminder_status NOT NULL DEFAULT 'scheduled',
    scheduled_for   TIMESTAMPTZ NOT NULL,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Budgets
CREATE TABLE budgets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id        UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    category_id         UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    limit_cents         BIGINT NOT NULL CHECK (limit_cents > 0),
    period              budget_period NOT NULL DEFAULT 'monthly',
    alert_threshold_pct SMALLINT NOT NULL DEFAULT 80
        CHECK (alert_threshold_pct BETWEEN 1 AND 100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, category_id, period)
);

CREATE TABLE budget_alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id       UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    spent_cents     BIGINT NOT NULL,
    limit_cents     BIGINT NOT NULL,
    threshold_pct   SMALLINT NOT NULL,
    status          alert_status NOT NULL DEFAULT 'active',
    triggered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ
);

-- Piggy banks
CREATE TABLE piggy_banks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    created_by      UUID NOT NULL REFERENCES users(id),
    name            VARCHAR(100) NOT NULL,
    target_cents    BIGINT NOT NULL CHECK (target_cents > 0),
    current_cents   BIGINT NOT NULL DEFAULT 0 CHECK (current_cents >= 0),
    target_date     DATE,
    status          piggy_bank_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE piggy_bank_contributions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    piggy_bank_id   UUID NOT NULL REFERENCES piggy_banks(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    note            TEXT,
    contributed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI budgeting (per household)
CREATE TABLE ai_budget_suggestions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    requested_by    UUID NOT NULL REFERENCES users(id),
    input_metadata  JSONB NOT NULL,
    suggestion      JSONB NOT NULL,
    model           VARCHAR(50) NOT NULL,
    tokens_used     INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notifications
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    household_id    UUID REFERENCES households(id) ON DELETE CASCADE,
    type            notification_type NOT NULL,
    channel         notification_channel NOT NULL DEFAULT 'in_app',
    title           VARCHAR(200) NOT NULL,
    body            TEXT NOT NULL,
    payload         JSONB,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Outbox for SSE
CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id    UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_expenses_household_date ON expenses(household_id, expense_date DESC);
CREATE INDEX idx_expense_splits_debtor ON expense_splits(debtor_id);
CREATE INDEX idx_household_balances_lookup ON household_balances(household_id, debtor_id);
CREATE INDEX idx_budgets_household ON budgets(household_id);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE read_at IS NULL;
CREATE INDEX idx_outbox_pending ON outbox_events(household_id) WHERE published_at IS NULL;
CREATE INDEX idx_debt_reminders_scheduled ON debt_reminders(scheduled_for) WHERE status = 'scheduled';
CREATE INDEX idx_ai_suggestions_household ON ai_budget_suggestions(household_id, created_at DESC);
