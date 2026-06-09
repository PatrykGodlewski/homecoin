package handlers

import (
	"net/http"

	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/dashboard"
)

func HandleDashboard(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}
	if sess.HouseholdID == "" {
		return kit.Redirect(http.StatusSeeOther, "/onboarding")
	}

	hh, err := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	if err != nil {
		return err
	}

	usage, err := appctx.App.UsageBudget.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	if err != nil {
		return err
	}

	expenses, _ := appctx.App.ListExpenses.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID, 100, 0)
	var totalSpent int64
	for _, e := range expenses {
		totalSpent += e.AmountCents
	}

	cups := make([]dashboard.CupData, len(usage))
	categories, _ := appctx.App.ListCategories.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	catNames := map[string]string{}
	for _, c := range categories {
		catNames[c.ID] = c.Name
	}

	for i, u := range usage {
		name := catNames[u.CategoryID]
		if name == "" {
			name = "Budget"
		}
		pct := int(u.UsagePercent)
		fillY, fillH := cupFill(pct)
		cups[i] = dashboard.CupData{
			Name:       name,
			Percent:    pct,
			Spent:      fmtMoney(u.SpentCents, hh.Household.Currency),
			Limit:      fmtMoney(u.LimitCents, hh.Household.Currency),
			OverBudget: u.ThresholdReached,
			FillY:      fillY,
			FillH:      fillH,
		}
	}

	banks, _ := appctx.App.ListPiggy.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	piggyBanks := make([]dashboard.PiggyBankData, len(banks))
	for i, pb := range banks {
		pct := 0
		if pb.TargetCents > 0 {
			pct = int(pb.CurrentCents * 100 / pb.TargetCents)
		}
		piggyBanks[i] = dashboard.PiggyBankData{
			Name:    pb.Name,
			Current: fmtMoney(pb.CurrentCents, hh.Household.Currency),
			Target:  fmtMoney(pb.TargetCents, hh.Household.Currency),
			Percent: pct,
			Status:  pb.Status,
		}
	}

	return kit.Render(dashboard.Overview(dashboard.OverviewData{
		UserName:    sess.DisplayName,
		Household:   hh.Household.Name,
		Currency:    hh.Household.Currency,
		MemberCount: len(hh.Members),
		TotalSpent:  fmtMoney(totalSpent, hh.Household.Currency),
		Cups:        cups,
		PiggyBanks:  piggyBanks,
	}))
}
