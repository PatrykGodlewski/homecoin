package appctx

import (
	"net/http"

	"github.com/anthdm/superkit/kit"
)

const sessionName = "homecoin_session"

type SessionAuth struct {
	UserID      string
	HouseholdID string
	DisplayName string
}

func (s SessionAuth) Check() bool { return s.UserID != "" }

func Authenticate(kit *kit.Kit) (kit.Auth, error) {
	sess := kit.GetSession(sessionName)
	userID, _ := sess.Values["user_id"].(string)
	householdID, _ := sess.Values["household_id"].(string)
	displayName, _ := sess.Values["display_name"].(string)
	return SessionAuth{UserID: userID, HouseholdID: householdID, DisplayName: displayName}, nil
}

func SetSession(kit *kit.Kit, userID, householdID, displayName string) error {
	sess := kit.GetSession(sessionName)
	sess.Values["user_id"] = userID
	sess.Values["household_id"] = householdID
	sess.Values["display_name"] = displayName
	return sess.Save(kit.Request, kit.Response)
}

func ClearSession(kit *kit.Kit) error {
	sess := kit.GetSession(sessionName)
	sess.Options.MaxAge = -1
	return sess.Save(kit.Request, kit.Response)
}

func CurrentAuth(kit *kit.Kit) SessionAuth {
	a := kit.Auth()
	if s, ok := a.(SessionAuth); ok {
		return s
	}
	return SessionAuth{}
}

func MustAuth(kit *kit.Kit) (SessionAuth, error) {
	auth := CurrentAuth(kit)
	if !auth.Check() {
		return auth, kit.Redirect(http.StatusSeeOther, "/login")
	}

	if App == nil || App.Me == nil {
		return auth, nil
	}

	me, err := App.Me.Execute(kit.Request.Context(), auth.UserID)
	if err != nil {
		_ = ClearSession(kit)
		return auth, kit.Redirect(http.StatusSeeOther, "/login")
	}

	householdID := ""
	if me.HouseholdID != nil {
		householdID = *me.HouseholdID
	}

	if householdID != auth.HouseholdID || me.DisplayName != auth.DisplayName {
		auth.HouseholdID = householdID
		auth.DisplayName = me.DisplayName
		_ = SetSession(kit, auth.UserID, householdID, me.DisplayName)
	}

	return auth, nil
}
