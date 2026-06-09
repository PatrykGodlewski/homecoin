package handlers

import (
	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/piggybank"
	piggybankuc "github.com/godlew/homecoin/internal/usecase/piggybank"
)

func HandlePiggyBanks(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	hh, _ := appctx.App.GetHH.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	banks, err := appctx.App.ListPiggy.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
	if err != nil {
		return err
	}

	items := make([]piggybank.Item, len(banks))
	for i, pb := range banks {
		pct := 0
		if pb.TargetCents > 0 {
			pct = int(pb.CurrentCents * 100 / pb.TargetCents)
		}
		items[i] = piggybank.Item{
			ID:       pb.ID,
			Name:     pb.Name,
			Current:  fmtMoney(pb.CurrentCents, hh.Household.Currency),
			Target:   fmtMoney(pb.TargetCents, hh.Household.Currency),
			Percent:  pct,
			Status:   pb.Status,
		}
	}

	return kit.Render(piggybank.ListPage(items))
}

func HandleCreatePiggyBank(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	_, err = appctx.App.CreatePiggy.Execute(kit.Request.Context(), piggybankuc.CreateInput{
		UserID:      sess.UserID,
		HouseholdID: sess.HouseholdID,
		Name:        kit.FormValue("name"),
		TargetCents: parseDollars(kit.FormValue("target")),
	})
	if err != nil {
		return err
	}
	return HandlePiggyBanks(kit)
}

func HandleContribute(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	_, err = appctx.App.Contribute.Execute(kit.Request.Context(), piggybankuc.ContributeInput{
		UserID:      sess.UserID,
		HouseholdID: sess.HouseholdID,
		PiggyBankID: URLParam(kit, "id"),
		AmountCents: parseDollars(kit.FormValue("amount")),
	})
	if err != nil {
		return err
	}
	return HandlePiggyBanks(kit)
}
