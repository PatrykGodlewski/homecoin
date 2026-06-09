package handlers

import (
	"net/http"

	"github.com/anthdm/superkit/kit"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/household"
	householduc "github.com/godlew/homecoin/internal/usecase/household"
)

func HandleOnboardingPage(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}
	if sess.HouseholdID != "" {
		return kit.Redirect(http.StatusSeeOther, "/dashboard")
	}
	return kit.Render(household.OnboardingPage(""))
}

func HandleCreateHousehold(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	out, err := appctx.App.CreateHH.Execute(kit.Request.Context(), householduc.CreateInput{
		UserID:   sess.UserID,
		Name:     kit.FormValue("name"),
		Currency: kit.FormValue("currency"),
	})
	if err != nil {
		return kit.Render(household.OnboardingPage(err.Error()))
	}

	if err := appctx.SetSession(kit, sess.UserID, out.HouseholdID, sess.DisplayName); err != nil {
		return err
	}
	return kit.Redirect(http.StatusSeeOther, "/dashboard")
}

func HandleJoinHousehold(kit *kit.Kit) error {
	sess, err := appctx.MustAuth(kit)
	if err != nil {
		return err
	}

	h, err := appctx.App.JoinHH.Execute(kit.Request.Context(), householduc.JoinInput{
		UserID:     sess.UserID,
		InviteCode: kit.FormValue("invite_code"),
	})
	if err != nil {
		return kit.Render(household.OnboardingPage(err.Error()))
	}

	if err := appctx.SetSession(kit, sess.UserID, h.ID, sess.DisplayName); err != nil {
		return err
	}
	return kit.Redirect(http.StatusSeeOther, "/dashboard")
}
