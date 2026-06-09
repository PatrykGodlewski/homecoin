package handlers

import (
	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/balance"
)

func HandleBalances(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	hh, _ := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	simplified, err := appctx.App.SimplifyBal.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	if err != nil {
		return err
	}

	names := map[string]string{}
	for _, m := range hh.Members {
		names[m.UserID] = m.DisplayName
	}

	items := make([]balance.Item, len(simplified))
	for i, b := range simplified {
		items[i] = balance.Item{
			From:   names[b.FromUserID],
			To:     names[b.ToUserID],
			Amount: fmtMoney(b.AmountCents, hh.Household.Currency),
		}
	}

	return kit.Render(balance.ListPage(items))
}
