package handlers

import (
	"fmt"

	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/expense"
	expenseuc "github.com/godlew/homecoin/internal/usecase/expense"
)

func HandleExpenses(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	hh, _ := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	list, err := appctx.App.ListExpenses.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID, 50, 0)
	if err != nil {
		return err
	}

	categories, _ := appctx.App.ListCategories.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	catNames := make(map[string]string, len(categories))
	catOpts := make([]expense.CategoryOption, len(categories))
	for i, c := range categories {
		catNames[c.ID] = c.Name
		catOpts[i] = expense.CategoryOption{ID: c.ID, Name: c.Name}
	}

	memberNames := make(map[string]string, len(hh.Members))
	members := make([]expense.Member, len(hh.Members))
	for i, m := range hh.Members {
		memberNames[m.UserID] = m.DisplayName
		members[i] = expense.Member{ID: m.UserID, Name: m.DisplayName}
	}

	items := make([]expense.Item, len(list))
	for i, e := range list {
		category := ""
		if e.CategoryID != nil {
			category = catNames[*e.CategoryID]
		}
		items[i] = expense.Item{
			Title:    e.Title,
			Amount:   fmtMoney(e.AmountCents, hh.Household.Currency),
			Split:    formatExpenseSplit(e, memberNames),
			Category: category,
		}
	}

	return kit.Render(expense.ListPage(expense.ListData{
		Items:      items,
		Members:    members,
		Categories: catOpts,
		Error:      "",
	}))
}

func HandleAddExpense(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	hh, _ := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	amount := parseDollars(kit.FormValue("amount"))
	payerID := kit.FormValue("payer_id")

	splitType := valueobject.SplitEqual
	splits := make([]valueobject.SplitInput, len(hh.Members))
	for i, m := range hh.Members {
		splits[i] = valueobject.SplitInput{DebtorID: m.UserID}
	}
	if debtorID := kit.FormValue("debtor_id"); debtorID != "" {
		splitType = valueobject.SplitExact
		splits = []valueobject.SplitInput{{DebtorID: debtorID, ExactCents: &amount}}
	}

	var categoryID *string
	if id := kit.FormValue("category_id"); id != "" {
		categoryID = &id
	}

	_, err = appctx.App.AddExpense.Execute(kit.Request.Context(), expenseuc.AddInput{
		HouseholdID: sess.HouseholdID,
		PayerID:     payerID,
		CreatedBy:   sess.UserID,
		Title:       kit.FormValue("title"),
		AmountCents: amount,
		SplitType:   splitType,
		SplitInputs: splits,
		CategoryID:  categoryID,
	})
	if err != nil {
		return HandleExpenses(kit)
	}
	return HandleExpenses(kit)
}

func formatExpenseSplit(e entity.Expense, memberNames map[string]string) string {
	if len(e.Splits) == 1 {
		if name, ok := memberNames[e.Splits[0].DebtorID]; ok {
			return name
		}
	}
	switch e.SplitType {
	case valueobject.SplitEqual:
		return "Equal"
	case valueobject.SplitExact:
		return "Exact"
	case valueobject.SplitPercentage:
		return "Percentage"
	case valueobject.SplitShares:
		return "Shares"
	default:
		return string(e.SplitType)
	}
}

func parseDollars(s string) int64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return int64(f * 100)
}
