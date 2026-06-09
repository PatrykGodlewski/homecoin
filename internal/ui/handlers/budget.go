package handlers

import (
	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/budget"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
)

func HandleBudgets(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	hh, _ := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	usage, err := appctx.App.UsageBudget.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	if err != nil {
		return err
	}

	categories, _ := appctx.App.ListCategories.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	catNames := map[string]string{}
	catOpts := make([]budget.CategoryOption, len(categories))
	for i, c := range categories {
		catNames[c.ID] = c.Name
		catOpts[i] = budget.CategoryOption{ID: c.ID, Name: c.Name}
	}

	items := make([]budget.Item, len(usage))
	for i, u := range usage {
		items[i] = budget.Item{
			Name:   catNames[u.CategoryID],
			Spent:  fmtMoney(u.SpentCents, hh.Household.Currency),
			Limit:  fmtMoney(u.LimitCents, hh.Household.Currency),
			Period: u.Period,
		}
	}

	return kit.Render(budget.ListPage(budget.ListData{Items: items, Categories: catOpts}))
}

func HandleCreateBudget(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	_, err = appctx.App.CreateBudget.Execute(kit.Request.Context(), budgetuc.CreateInput{
		UserID:            sess.UserID,
		HouseholdID:       sess.HouseholdID,
		CategoryID:        kit.FormValue("category_id"),
		LimitCents:        parseDollars(kit.FormValue("limit")),
		Period:            "monthly",
		AlertThresholdPct: 80,
	})
	if err != nil {
		return err
	}
	return HandleBudgets(kit)
}
