package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
)

func (r *UserRepo) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET display_name = $2, avatar_url = $3, monthly_income_cents = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL`,
		user.ID, user.DisplayName, user.AvatarURL, user.MonthlyIncomeCents, user.UpdatedAt,
	)
	return err
}

func (r *HouseholdRepo) RemoveMember(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM household_members WHERE user_id = $1`, userID)
	return err
}

func (r *HouseholdRepo) ListAllIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM households WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SettlementRepo) GetByID(ctx context.Context, id string) (*entity.Settlement, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, household_id, from_user_id, to_user_id, amount_cents, status, note, settled_at, created_at, updated_at
		FROM settlements WHERE id = $1`, id)

	var s entity.Settlement
	err := row.Scan(&s.ID, &s.HouseholdID, &s.FromUserID, &s.ToUserID, &s.AmountCents, &s.Status, &s.Note, &s.SettledAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *BudgetRepo) GetByID(ctx context.Context, id string) (*entity.Budget, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, household_id, category_id, limit_cents, period, alert_threshold_pct, created_at, updated_at
		FROM budgets WHERE id = $1`, id)

	var b entity.Budget
	err := row.Scan(&b.ID, &b.HouseholdID, &b.CategoryID, &b.LimitCents, &b.Period, &b.AlertThresholdPct, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *BudgetRepo) ListDistinctHouseholdIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT household_id FROM budgets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *CategoryRepo) GetByID(ctx context.Context, id string) (*entity.Category, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, household_id, name, icon, color, is_fixed, created_at, updated_at
		FROM categories WHERE id = $1`, id)

	var c entity.Category
	err := row.Scan(&c.ID, &c.HouseholdID, &c.Name, &c.Icon, &c.Color, &c.IsFixed, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *PiggyBankRepo) GetByID(ctx context.Context, id string) (*entity.PiggyBank, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, household_id, created_by, name, target_cents, current_cents, target_date, status, created_at, updated_at
		FROM piggy_banks WHERE id = $1`, id)

	var pb entity.PiggyBank
	err := row.Scan(&pb.ID, &pb.HouseholdID, &pb.CreatedBy, &pb.Name, &pb.TargetCents, &pb.CurrentCents, &pb.TargetDate, &pb.Status, &pb.CreatedAt, &pb.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &pb, nil
}

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

func (r *NotificationRepo) Create(ctx context.Context, n *entity.Notification) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	n.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, household_id, type, channel, title, body, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.UserID, n.HouseholdID, n.Type, n.Channel, n.Title, n.Body, n.Payload, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID string, unreadOnly bool, limit int32) ([]entity.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, user_id, household_id, type, channel, title, body, payload, read_at, created_at
		FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND read_at IS NULL`
	}
	query += ` ORDER BY created_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotifications(rows)
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func scanNotifications(rows pgx.Rows) ([]entity.Notification, error) {
	var list []entity.Notification
	for rows.Next() {
		var n entity.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.HouseholdID, &n.Type, &n.Channel, &n.Title, &n.Body, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

type DebtReminderRepo struct {
	pool *pgxpool.Pool
}

func NewDebtReminderRepo(pool *pgxpool.Pool) *DebtReminderRepo {
	return &DebtReminderRepo{pool: pool}
}

func (r *DebtReminderRepo) Create(ctx context.Context, rem *entity.DebtReminder) error {
	if rem.ID == "" {
		rem.ID = uuid.NewString()
	}
	rem.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO debt_reminders (id, household_id, creditor_id, debtor_id, amount_cents, status, scheduled_for, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		rem.ID, rem.HouseholdID, rem.CreditorID, rem.DebtorID, rem.AmountCents, rem.Status, rem.ScheduledFor, rem.CreatedAt,
	)
	return err
}

func (r *DebtReminderRepo) ListScheduled(ctx context.Context, before time.Time, limit int32) ([]entity.DebtReminder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, creditor_id, debtor_id, amount_cents, status, scheduled_for, sent_at, created_at
		FROM debt_reminders
		WHERE status = 'scheduled' AND scheduled_for <= $1
		ORDER BY scheduled_for ASC LIMIT $2`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReminders(rows)
}

func (r *DebtReminderRepo) MarkSent(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE debt_reminders SET status = 'sent', sent_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *DebtReminderRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.DebtReminder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, creditor_id, debtor_id, amount_cents, status, scheduled_for, sent_at, created_at
		FROM debt_reminders WHERE household_id = $1 ORDER BY created_at DESC`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReminders(rows)
}

func scanReminders(rows pgx.Rows) ([]entity.DebtReminder, error) {
	var list []entity.DebtReminder
	for rows.Next() {
		var rem entity.DebtReminder
		if err := rows.Scan(&rem.ID, &rem.HouseholdID, &rem.CreditorID, &rem.DebtorID, &rem.AmountCents, &rem.Status, &rem.ScheduledFor, &rem.SentAt, &rem.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rem)
	}
	return list, rows.Err()
}

type BudgetAlertRepo struct {
	pool *pgxpool.Pool
}

func NewBudgetAlertRepo(pool *pgxpool.Pool) *BudgetAlertRepo {
	return &BudgetAlertRepo{pool: pool}
}

func (r *BudgetAlertRepo) Create(ctx context.Context, a *entity.BudgetAlert) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.TriggeredAt.IsZero() {
		a.TriggeredAt = time.Now()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO budget_alerts (id, budget_id, household_id, spent_cents, limit_cents, threshold_pct, status, triggered_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		a.ID, a.BudgetID, a.HouseholdID, a.SpentCents, a.LimitCents, a.ThresholdPct, a.Status, a.TriggeredAt,
	)
	return err
}

func (r *BudgetAlertRepo) HasActiveAlert(ctx context.Context, budgetID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM budget_alerts WHERE budget_id = $1 AND status = 'active'
		)`, budgetID).Scan(&exists)
	return exists, err
}

func (r *BudgetAlertRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.BudgetAlert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, budget_id, household_id, spent_cents, limit_cents, threshold_pct, status, triggered_at, acknowledged_at
		FROM budget_alerts WHERE household_id = $1 ORDER BY triggered_at DESC`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []entity.BudgetAlert
	for rows.Next() {
		var a entity.BudgetAlert
		if err := rows.Scan(&a.ID, &a.BudgetID, &a.HouseholdID, &a.SpentCents, &a.LimitCents, &a.ThresholdPct, &a.Status, &a.TriggeredAt, &a.AcknowledgedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *BudgetAlertRepo) Acknowledge(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE budget_alerts SET status = 'acknowledged', acknowledged_at = NOW()
		WHERE id = $1 AND status = 'active'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrNotFound
	}
	_ = userID
	return nil
}

func (r *PiggyBankRepo) GetAfterContribution(ctx context.Context, id string) (*entity.PiggyBank, error) {
	return r.GetByID(ctx, id)
}
