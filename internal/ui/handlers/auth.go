package handlers

import (
	"net/http"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/views/auth"
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
)

func HandleLoginPage(kit *kit.Kit) error {
	return kit.Render(auth.LoginPage(""))
}

func HandleRegisterPage(kit *kit.Kit) error {
	return kit.Render(auth.RegisterPage(""))
}

func HandleLogin(kit *kit.Kit) error {
	email := kit.FormValue("email")
	password := kit.FormValue("password")

	out, err := appctx.App.Login.Execute(kit.Request.Context(), authuc.LoginInput{Email: email, Password: password})
	if err != nil {
		return kit.Render(auth.LoginPage("Invalid email or password"))
	}

	me, _ := appctx.App.Me.Execute(kit.Request.Context(), out.UserID)
	name := email
	hhID := ""
	if me != nil {
		name = me.DisplayName
		if me.HouseholdID != nil {
			hhID = *me.HouseholdID
		}
	}

	if err := appctx.SetSession(kit, out.UserID, hhID, name); err != nil {
		return err
	}

	if hhID == "" {
		return kit.Redirect(http.StatusSeeOther, "/onboarding")
	}
	return kit.Redirect(http.StatusSeeOther, "/dashboard")
}

func HandleRegister(kit *kit.Kit) error {
	out, err := appctx.App.Register.Execute(kit.Request.Context(), authuc.RegisterInput{
		Email:       kit.FormValue("email"),
		Password:    kit.FormValue("password"),
		DisplayName: kit.FormValue("display_name"),
	})
	if err != nil {
		return kit.Render(auth.RegisterPage(err.Error()))
	}

	if err := appctx.SetSession(kit, out.UserID, "", kit.FormValue("display_name")); err != nil {
		return err
	}
	return kit.Redirect(http.StatusSeeOther, "/onboarding")
}

func HandleLogout(kit *kit.Kit) error {
	_ = appctx.ClearSession(kit)
	return kit.Redirect(http.StatusSeeOther, "/login")
}

// chi URL param helper for piggy bank contribute
func URLParam(kit *kit.Kit, key string) string {
	return chi.URLParam(kit.Request, key)
}
