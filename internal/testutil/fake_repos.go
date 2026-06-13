package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
)

type FakeUserRepo struct {
	Users map[string]*entity.User
	ByEmail map[string]string
}

func NewFakeUserRepo() *FakeUserRepo {
	return &FakeUserRepo{
		Users:   make(map[string]*entity.User),
		ByEmail: make(map[string]string),
	}
}

func (r *FakeUserRepo) Create(ctx context.Context, user *entity.User) error {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	r.Users[user.ID] = user
	r.ByEmail[user.Email.String()] = user.ID
	return nil
}

func (r *FakeUserRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	u, ok := r.Users[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return u, nil
}

func (r *FakeUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	id, ok := r.ByEmail[email]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return r.Users[id], nil
}

func (r *FakeUserRepo) Update(ctx context.Context, user *entity.User) error {
	if _, ok := r.Users[user.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	user.UpdatedAt = time.Now()
	r.Users[user.ID] = user
	return nil
}

type FakeRefreshTokenRepo struct {
	Created []struct {
		UserID    string
		TokenHash string
		ExpiresAt time.Time
	}
}

func (r *FakeRefreshTokenRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	r.Created = append(r.Created, struct {
		UserID    string
		TokenHash string
		ExpiresAt time.Time
	}{userID, tokenHash, expiresAt})
	return nil
}

func (r *FakeRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (string, time.Time, *time.Time, error) {
	return "", time.Time{}, nil, domainerrors.ErrNotFound
}

func (r *FakeRefreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	return nil
}

type FakeHouseholdRepo struct {
	Households map[string]*entity.Household
	Members    map[string]*entity.HouseholdMember
	ByInvite   map[string]string
}

func NewFakeHouseholdRepo() *FakeHouseholdRepo {
	return &FakeHouseholdRepo{
		Households: make(map[string]*entity.Household),
		Members:    make(map[string]*entity.HouseholdMember),
		ByInvite:   make(map[string]string),
	}
}

func (r *FakeHouseholdRepo) Create(ctx context.Context, h *entity.Household) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	now := time.Now()
	h.CreatedAt = now
	h.UpdatedAt = now
	r.Households[h.ID] = h
	if h.InviteCode != nil {
		r.ByInvite[*h.InviteCode] = h.ID
	}
	return nil
}

func (r *FakeHouseholdRepo) GetByID(ctx context.Context, id string) (*entity.Household, error) {
	h, ok := r.Households[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return h, nil
}

func (r *FakeHouseholdRepo) GetByInviteCode(ctx context.Context, code string) (*entity.Household, error) {
	id, ok := r.ByInvite[code]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return r.Households[id], nil
}

func (r *FakeHouseholdRepo) AddMember(ctx context.Context, member *entity.HouseholdMember) error {
	if member.ID == "" {
		member.ID = uuid.NewString()
	}
	member.JoinedAt = time.Now()
	r.Members[member.UserID] = member
	return nil
}

func (r *FakeHouseholdRepo) RemoveMember(ctx context.Context, userID string) error {
	delete(r.Members, userID)
	return nil
}

func (r *FakeHouseholdRepo) GetMemberByUserID(ctx context.Context, userID string) (*entity.HouseholdMember, error) {
	m, ok := r.Members[userID]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return m, nil
}

func (r *FakeHouseholdRepo) GetMembers(ctx context.Context, householdID string) ([]entity.HouseholdMember, error) {
	var out []entity.HouseholdMember
	for _, m := range r.Members {
		if m.HouseholdID == householdID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (r *FakeHouseholdRepo) ListAllIDs(ctx context.Context) ([]string, error) {
	ids := make([]string, 0, len(r.Households))
	for id := range r.Households {
		ids = append(ids, id)
	}
	return ids, nil
}

type FakeCategoryRepo struct {
	Categories []*entity.Category
}

func (r *FakeCategoryRepo) Create(ctx context.Context, c *entity.Category) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	r.Categories = append(r.Categories, c)
	return nil
}

func (r *FakeCategoryRepo) GetByID(ctx context.Context, id string) (*entity.Category, error) {
	for _, c := range r.Categories {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}

func (r *FakeCategoryRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.Category, error) {
	var out []entity.Category
	for _, c := range r.Categories {
		if c.HouseholdID == householdID {
			out = append(out, *c)
		}
	}
	return out, nil
}

type FakeExpenseRepo struct {
	Expenses []*entity.Expense
}

func (r *FakeExpenseRepo) Create(ctx context.Context, expense *entity.Expense) error {
	if expense.ID == "" {
		expense.ID = uuid.NewString()
	}
	now := time.Now()
	expense.CreatedAt = now
	expense.UpdatedAt = now
	r.Expenses = append(r.Expenses, expense)
	return nil
}

func (r *FakeExpenseRepo) ListByHousehold(ctx context.Context, householdID string, limit, offset int32) ([]entity.Expense, error) {
	var out []entity.Expense
	for _, e := range r.Expenses {
		if e.HouseholdID == householdID {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (r *FakeExpenseRepo) ListAllByHousehold(ctx context.Context, householdID string) ([]entity.Expense, error) {
	return r.ListByHousehold(ctx, householdID, 0, 0)
}

func (r *FakeExpenseRepo) GetCategorySpend(ctx context.Context, householdID, categoryID string, from, to time.Time) (int64, error) {
	return 0, nil
}

type FakeOutboxRepo struct {
	Events []*entity.OutboxEvent
}

func (r *FakeOutboxRepo) Insert(ctx context.Context, event *entity.OutboxEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	r.Events = append(r.Events, event)
	return nil
}

func (r *FakeOutboxRepo) FetchPending(ctx context.Context, limit int32) ([]entity.OutboxEvent, error) {
	return nil, nil
}

func (r *FakeOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	return nil
}

type FakeBalanceRepo struct {
	Balances []entity.HouseholdBalance
	Upserts  []repository.BalancePair
}

func (r *FakeBalanceRepo) UpsertBatch(ctx context.Context, householdID string, pairs []repository.BalancePair) error {
	r.Upserts = pairs
	return nil
}

func (r *FakeBalanceRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.HouseholdBalance, error) {
	return r.Balances, nil
}

type FakeSettlementRepo struct {
	Settlements map[string]*entity.Settlement
}

func NewFakeSettlementRepo() *FakeSettlementRepo {
	return &FakeSettlementRepo{Settlements: make(map[string]*entity.Settlement)}
}

func (r *FakeSettlementRepo) Create(ctx context.Context, s *entity.Settlement) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	r.Settlements[s.ID] = s
	return nil
}

func (r *FakeSettlementRepo) GetByID(ctx context.Context, id string) (*entity.Settlement, error) {
	s, ok := r.Settlements[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return s, nil
}

func (r *FakeSettlementRepo) UpdateStatus(ctx context.Context, id, status string) error {
	s, ok := r.Settlements[id]
	if !ok {
		return domainerrors.ErrNotFound
	}
	s.Status = status
	return nil
}

func (r *FakeSettlementRepo) ListByHousehold(ctx context.Context, householdID string) ([]entity.Settlement, error) {
	var out []entity.Settlement
	for _, s := range r.Settlements {
		if s.HouseholdID == householdID {
			out = append(out, *s)
		}
	}
	return out, nil
}

type FakeNotificationRepo struct {
	Created []*entity.Notification
}

func (r *FakeNotificationRepo) Create(ctx context.Context, n *entity.Notification) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	r.Created = append(r.Created, n)
	return nil
}

func (r *FakeNotificationRepo) ListByUser(ctx context.Context, userID string, unreadOnly bool, limit int32) ([]entity.Notification, error) {
	return nil, nil
}

func (r *FakeNotificationRepo) MarkRead(ctx context.Context, id, userID string) error {
	return nil
}

type RecalcSpy struct {
	mu    sync.Mutex
	Calls []string
}

func (s *RecalcSpy) Trigger(ctx context.Context, householdID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = append(s.Calls, householdID)
}

func (s *RecalcSpy) HouseholdIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.Calls))
	copy(out, s.Calls)
	return out
}
