package httpapi_test

import (
	"net/http"
	"testing"
)

func TestTeams_CreateUpdateListAndDelete(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/teams", ""), http.StatusUnauthorized)
	api := signedIn(t, "Ash")
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	list := decode[struct{ Pokemon []struct{ ID string } }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	first, second := list.Pokemon[0].ID, list.Pokemon[1].ID

	rec := api.do(http.MethodPost, "/api/v1/teams", `{"name":"Raid","memberIds":["`+first+`"]}`)
	wantStatus(t, rec, http.StatusCreated)
	team := decode[struct {
		ID        string
		Name      string
		MemberIDs []string `json:"memberIds"`
	}](t, rec.Body.Bytes())

	rec = api.do(http.MethodPost, "/api/v1/teams", `{"name":"raid","memberIds":[]}`)
	wantStatus(t, rec, http.StatusConflict)
	if code := errorCode(t, rec); code != "TEAM_NAME_TAKEN" {
		t.Fatalf("code = %q", code)
	}
	wantStatus(t, api.do(http.MethodPost, "/api/v1/teams", `{"name":"GBL","memberIds":["missing"]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/teams", `{"name":"   ","memberIds":[]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/teams/nope", `{"name":"X","memberIds":[]}`), http.StatusNotFound)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/teams/"+team.ID, `{"name":"Raid team","memberIds":["`+first+`","`+second+`"]}`), http.StatusOK)

	teams := decode[struct {
		Teams []struct {
			Name      string
			MemberIDs []string `json:"memberIds"`
		}
	}](t, api.do(http.MethodGet, "/api/v1/teams", "").Body.Bytes())
	if len(teams.Teams) != 1 || teams.Teams[0].Name != "Raid team" || len(teams.Teams[0].MemberIDs) != 2 {
		t.Fatalf("teams = %+v", teams)
	}

	wantStatus(t, api.do(http.MethodDelete, "/api/v1/teams/"+team.ID, ""), http.StatusNoContent)
	wantStatus(t, api.do(http.MethodDelete, "/api/v1/teams/"+team.ID, ""), http.StatusNotFound)
}
