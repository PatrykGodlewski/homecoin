package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/valueobject"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, monthly_income_cents, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email.String(), user.PasswordHash, user.DisplayName, user.MonthlyIncomeCents, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, avatar_url, monthly_income_cents, email_verified, created_at, updated_at
		FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, avatar_url, monthly_income_cents, email_verified, created_at, updated_at
		FROM users WHERE email = $1 AND deleted_at IS NULL`, email)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*entity.User, error) {
	var u entity.User
	var email string
	err := row.Scan(&u.ID, &email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.MonthlyIncomeCents, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Email = valueobject.Email(email)
	return &u, nil
}

type HouseholdRepo struct {
	pool *pgxpool.Pool
}

func NewHouseholdRepo(pool *pgxpool.Pool) *HouseholdRepo {
	return &HouseholdRepo{pool: pool}
}

func (r *HouseholdRepo) Create(ctx context.Context, h *entity.Household) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	now := time.Now()
	h.CreatedAt = now
	h.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO households (id, name, currency, invite_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		h.ID, h.Name, h.Currency, h.InviteCode, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert household: %w", err)
	}
	return nil
}

func (r *HouseholdRepo) GetByID(ctx context.Context, id string) (*entity.Household, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, currency, invite_code, created_at, updated_at
		FROM households WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanHousehold(row)
}

func (r *HouseholdRepo) GetByInviteCode(ctx context.Context, code string) (*entity.Household, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, currency, invite_code, created_at, updated_at
		FROM households WHERE invite_code = $1 AND deleted_at IS NULL`, code)
	return scanHousehold(row)
}

func scanHousehold(row pgx.Row) (*entity.Household, error) {
	var h entity.Household
	err := row.Scan(&h.ID, &h.Name, &h.Currency, &h.InviteCode, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan household: %w", err)
	}
	return &h, nil
}

func (r *HouseholdRepo) AddMember(ctx context.Context, m *entity.HouseholdMember) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now()
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO household_members (id, household_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)`,
		m.ID, m.HouseholdID, m.UserID, m.Role, m.JoinedAt,
	)
	if err != nil {
		return fmt.Errorf("insert household member: %w", err)
	}
	return nil
}

func (r *HouseholdRepo) GetMemberByUserID(ctx context.Context, userID string) (*entity.HouseholdMember, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, household_id, user_id, role, joined_at
		FROM household_members WHERE user_id = $1`, userID)

	var m entity.HouseholdMember
	err := row.Scan(&m.ID, &m.HouseholdID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan member: %w", err)
	}
	return &m, nil
}

func (r *HouseholdRepo) GetMembers(ctx context.Context, householdID string) ([]entity.HouseholdMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, user_id, role, joined_at
		FROM household_members WHERE household_id = $1`, householdID)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	var members []entity.HouseholdMember
	for rows.Next() {
		var m entity.HouseholdMember
		if err := rows.Scan(&m.ID, &m.HouseholdID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member row: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		uuid.NewString(), userID, tokenHash, expiresAt,
	)
	return err
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (string, time.Time, *time.Time, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = $1`, tokenHash)

	var userID string
	var expiresAt time.Time
	var revokedAt *time.Time
	if err := row.Scan(&userID, &expiresAt, &revokedAt); err != nil {
		if err == pgx.ErrNoRows {
			return "", time.Time{}, nil, domainerrors.ErrNotFound
		}
		return "", time.Time{}, nil, err
	}
	return userID, expiresAt, revokedAt, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}

type ExpenseRepo struct {
	pool *pgxpool.Pool
}

func NewExpenseRepo(pool *pgxpool.Pool) *ExpenseRepo {
	return &ExpenseRepo{pool: pool}
}

func (r *ExpenseRepo) Create(ctx context.Context, expense *entity.Expense) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if expense.ID == "" {
		expense.ID = uuid.NewString()
	}
	now := time.Now()
	expense.CreatedAt = now
	expense.UpdatedAt = now

	_, err = tx.Exec(ctx, `
		INSERT INTO expenses (id, household_id, payer_id, category_id, title, description, amount_cents, split_type, expense_date, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		expense.ID, expense.HouseholdID, expense.PayerID, expense.CategoryID, expense.Title,
		expense.Description, expense.AmountCents, expense.SplitType, expense.ExpenseDate,
		expense.CreatedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert expense: %w", err)
	}

	for i := range expense.Splits {
		split := &expense.Splits[i]
		if split.ID == "" {
			split.ID = uuid.NewString()
		}
		split.ExpenseID = expense.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO expense_splits (id, expense_id, debtor_id, amount_cents, exact_amount_cents, percentage, shares)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			split.ID, split.ExpenseID, split.DebtorID, split.AmountCents,
			split.ExactAmountCents, split.Percentage, split.Shares,
		)
		if err != nil {
			return fmt.Errorf("insert split: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ExpenseRepo) ListByHousehold(ctx context.Context, householdID string, limit, offset int32) ([]entity.Expense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, payer_id, category_id, title, description, amount_cents, split_type, expense_date, created_by, created_at, updated_at
		FROM expenses WHERE household_id = $1 AND deleted_at IS NULL
		ORDER BY expense_date DESC, created_at DESC LIMIT $2 OFFSET $3`,
		householdID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanExpensesWithSplits(ctx, r.pool, rows)
}

func (r *ExpenseRepo) ListAllByHousehold(ctx context.Context, householdID string) ([]entity.Expense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, payer_id, category_id, title, description, amount_cents, split_type, expense_date, created_by, created_at, updated_at
		FROM expenses WHERE household_id = $1 AND deleted_at IS NULL
		ORDER BY expense_date DESC`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanExpensesWithSplits(ctx, r.pool, rows)
}

func scanExpensesWithSplits(ctx context.Context, pool *pgxpool.Pool, rows pgx.Rows) ([]entity.Expense, error) {
	var expenses []entity.Expense
	var ids []string

	for rows.Next() {
		var e entity.Expense
		if err := rows.Scan(&e.ID, &e.HouseholdID, &e.PayerID, &e.CategoryID, &e.Title, &e.Description,
			&e.AmountCents, &e.SplitType, &e.ExpenseDate, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
		ids = append(ids, e.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return expenses, nil
	}

	splitRows, err := pool.Query(ctx, `
		SELECT id, expense_id, debtor_id, amount_cents, exact_amount_cents, percentage, shares
		FROM expense_splits WHERE expense_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer splitRows.Close()

	splitsByExpense := make(map[string][]entity.ExpenseSplit)
	for splitRows.Next() {
		var s entity.ExpenseSplit
		if err := splitRows.Scan(&s.ID, &s.ExpenseID, &s.DebtorID, &s.AmountCents, &s.ExactAmountCents, &s.Percentage, &s.Shares); err != nil {
			return nil, err
		}
		splitsByExpense[s.ExpenseID] = append(splitsByExpense[s.ExpenseID], s)
	}

	for i := range expenses {
		expenses[i].Splits = splitsByExpense[expenses[i].ID]
	}
	return expenses, splitRows.Err()
}

func (r *ExpenseRepo) GetCategorySpend(ctx context.Context, householdID, categoryID string, from, to time.Time) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0)::bigint
		FROM expenses
		WHERE household_id = $1 AND category_id = $2 AND deleted_at IS NULL
		  AND expense_date >= $3 AND expense_date <= $4`,
		householdID, categoryID, from, to,
	).Scan(&total)
	return total, err
}

type BalanceRepo struct {
	pool *pgxpool.Pool
}

func NewBalanceRepo(pool *pgxpool.Pool) *BalanceRepo {
	return &BalanceRepo{pool: pool}
}

func (r *BalanceRepo) UpsertBatch(ctx context.Context, householdID string, pairs []repository.BalancePair) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM household_balances WHERE household_id = $1`, householdID)
	if err != nil {
		return err
	}

	for _, p := range pairs {
		_, err = tx.Exec(ctx, `
			INSERT INTO household_balances (id, household_id, creditor_id, debtor_id, balance_cents, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())`,
			uuid.NewString(), householdID, p.CreditorID, p.DebtorID, p.BalanceCents,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *BalanceRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.HouseholdBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, creditor_id, debtor_id, balance_cents, updated_at
		FROM household_balances WHERE household_id = $1 AND balance_cents <> 0`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []entity.HouseholdBalance
	for rows.Next() {
		var b entity.HouseholdBalance
		if err := rows.Scan(&b.ID, &b.HouseholdID, &b.CreditorID, &b.DebtorID, &b.BalanceCents, &b.UpdatedAt); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

type SettlementRepo struct {
	pool *pgxpool.Pool
}

func NewSettlementRepo(pool *pgxpool.Pool) *SettlementRepo {
	return &SettlementRepo{pool: pool}
}

func (r *SettlementRepo) Create(ctx context.Context, s *entity.Settlement) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	return r.pool.QueryRow(ctx, `
		INSERT INTO settlements (id, household_id, from_user_id, to_user_id, amount_cents, status, note, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`, s.ID, s.HouseholdID, s.FromUserID, s.ToUserID, s.AmountCents, s.Status, s.Note, now, now,
	).Scan(&s.ID)
}

func (r *SettlementRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE settlements SET status = $2,
			settled_at = CASE WHEN $2 = 'confirmed' THEN NOW() ELSE settled_at END,
			updated_at = NOW()
		WHERE id = $1`, id, status)
	return err
}

func (r *SettlementRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.Settlement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, from_user_id, to_user_id, amount_cents, status, note, settled_at, created_at, updated_at
		FROM settlements WHERE household_id = $1 ORDER BY created_at DESC`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settlements []entity.Settlement
	for rows.Next() {
		var s entity.Settlement
		if err := rows.Scan(&s.ID, &s.HouseholdID, &s.FromUserID, &s.ToUserID, &s.AmountCents, &s.Status, &s.Note, &s.SettledAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settlements = append(settlements, s)
	}
	return settlements, rows.Err()
}

type OutboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

func (r *OutboxRepo) Insert(ctx context.Context, event *entity.OutboxEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	event.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO outbox_events (id, household_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		event.ID, event.HouseholdID, event.EventType, event.Payload, event.CreatedAt,
	)
	return err
}

func (r *OutboxRepo) FetchPending(ctx context.Context, limit int32) ([]entity.OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, event_type, payload, published_at, created_at
		FROM outbox_events WHERE published_at IS NULL
		ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []entity.OutboxEvent
	for rows.Next() {
		var e entity.OutboxEvent
		if err := rows.Scan(&e.ID, &e.HouseholdID, &e.EventType, &e.Payload, &e.PublishedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox_events SET published_at = NOW() WHERE id = $1`, id)
	return err
}

type BudgetRepo struct {
	pool *pgxpool.Pool
}

func NewBudgetRepo(pool *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{pool: pool}
}

func (r *BudgetRepo) Create(ctx context.Context, b *entity.Budget) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO budgets (id, household_id, category_id, limit_cents, period, alert_threshold_pct, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, b.HouseholdID, b.CategoryID, b.LimitCents, b.Period, b.AlertThresholdPct, now, now,
	)
	return err
}

func (r *BudgetRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.Budget, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, category_id, limit_cents, period, alert_threshold_pct, created_at, updated_at
		FROM budgets WHERE household_id = $1`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []entity.Budget
	for rows.Next() {
		var b entity.Budget
		if err := rows.Scan(&b.ID, &b.HouseholdID, &b.CategoryID, &b.LimitCents, &b.Period, &b.AlertThresholdPct, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

type CategoryRepo struct {
	pool *pgxpool.Pool
}

func NewCategoryRepo(pool *pgxpool.Pool) *CategoryRepo {
	return &CategoryRepo{pool: pool}
}

func (r *CategoryRepo) Create(ctx context.Context, c *entity.Category) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO categories (id, household_id, name, icon, color, is_fixed, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.HouseholdID, c.Name, c.Icon, c.Color, c.IsFixed, now, now,
	)
	return err
}

func (r *CategoryRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, icon, color, is_fixed, created_at, updated_at
		FROM categories WHERE household_id = $1 ORDER BY name`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []entity.Category
	for rows.Next() {
		var c entity.Category
		if err := rows.Scan(&c.ID, &c.HouseholdID, &c.Name, &c.Icon, &c.Color, &c.IsFixed, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

type AISuggestionRepo struct {
	pool *pgxpool.Pool
}

func NewAISuggestionRepo(pool *pgxpool.Pool) *AISuggestionRepo {
	return &AISuggestionRepo{pool: pool}
}

func (r *AISuggestionRepo) Create(ctx context.Context, s *entity.AIBudgetSuggestion) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_budget_suggestions (id, household_id, requested_by, input_metadata, suggestion, model, tokens_used, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		s.ID, s.HouseholdID, s.RequestedBy, s.InputMetadata, s.Suggestion, s.Model, s.TokensUsed, s.CreatedAt,
	)
	return err
}

func (r *AISuggestionRepo) ListByHousehold(ctx context.Context, householdID string, limit int32) ([]entity.AIBudgetSuggestion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, requested_by, input_metadata, suggestion, model, tokens_used, created_at
		FROM ai_budget_suggestions WHERE household_id = $1
		ORDER BY created_at DESC LIMIT $2`, householdID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []entity.AIBudgetSuggestion
	for rows.Next() {
		var s entity.AIBudgetSuggestion
		if err := rows.Scan(&s.ID, &s.HouseholdID, &s.RequestedBy, &s.InputMetadata, &s.Suggestion, &s.Model, &s.TokensUsed, &s.CreatedAt); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, rows.Err()
}

type PiggyBankRepo struct {
	pool *pgxpool.Pool
}

func NewPiggyBankRepo(pool *pgxpool.Pool) *PiggyBankRepo {
	return &PiggyBankRepo{pool: pool}
}

func (r *PiggyBankRepo) Create(ctx context.Context, pb *entity.PiggyBank) error {
	if pb.ID == "" {
		pb.ID = uuid.NewString()
	}
	now := time.Now()
	pb.CreatedAt = now
	pb.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO piggy_banks (id, household_id, created_by, name, target_cents, current_cents, target_date, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		pb.ID, pb.HouseholdID, pb.CreatedBy, pb.Name, pb.TargetCents, pb.CurrentCents, pb.TargetDate, pb.Status, now, now,
	)
	return err
}

func (r *PiggyBankRepo) AddContribution(ctx context.Context, piggyBankID, userID string, amountCents int64, note *string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO piggy_bank_contributions (id, piggy_bank_id, user_id, amount_cents, note)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.NewString(), piggyBankID, userID, amountCents, note,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE piggy_banks SET current_cents = current_cents + $2, updated_at = NOW(),
			status = CASE WHEN current_cents + $2 >= target_cents THEN 'completed'::piggy_bank_status ELSE status END
		WHERE id = $1`, piggyBankID, amountCents)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PiggyBankRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.PiggyBank, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, created_by, name, target_cents, current_cents, target_date, status, created_at, updated_at
		FROM piggy_banks WHERE household_id = $1 ORDER BY created_at DESC`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []entity.PiggyBank
	for rows.Next() {
		var pb entity.PiggyBank
		if err := rows.Scan(&pb.ID, &pb.HouseholdID, &pb.CreatedBy, &pb.Name, &pb.TargetCents, &pb.CurrentCents, &pb.TargetDate, &pb.Status, &pb.CreatedAt, &pb.UpdatedAt); err != nil {
			return nil, err
		}
		banks = append(banks, pb)
	}
	return banks, rows.Err()
}

// Helper to marshal outbox payloads
func MarshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}
