package httpapi_test

import (
	"net/http"
	"testing"
)

func TestKeep_SetupThenSummary(t *testing.T) {
	api := signedIn(t, "Ash")
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)

	wantStatus(t, api.do(http.MethodGet, "/api/v1/setup", ""), http.StatusNotFound)
	rec := api.do(http.MethodGet, "/api/v1/keep-summary", "")
	wantStatus(t, rec, http.StatusConflict)
	if code := errorCode(t, rec); code != "SETUP_REQUIRED" {
		t.Fatalf("code = %q", code)
	}

	wantStatus(t, api.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodGet, "/api/v1/setup", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"features":["shiny","costume","background","dynamax","gigantamax"]}`), http.StatusOK)

	summary := decode[struct {
		Total, Kept, Surplus, NeedsConfirmation, Minimum int
		Groups                                           []struct {
			Species    string
			Keep       int
			SurplusIDs []string `json:"surplusIds"`
		}
	}](t, api.do(http.MethodGet, "/api/v1/keep-summary", "").Body.Bytes())
	if summary.Total != 4 || summary.Minimum != 1 || summary.NeedsConfirmation != 0 || summary.Kept+summary.Surplus != 4 || len(summary.Groups) == 0 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestKeep_RejectsInvalidSetupAndAnonymousRequests(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/keep-summary", ""), http.StatusUnauthorized)
	wantStatus(t, newAPI(t).do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusUnauthorized)

	api := signedIn(t, "Ash")
	for name, body := range map[string]string{
		"no goals":      `{"capacity":350,"goals":[]}`,
		"repeated goal": `{"capacity":350,"goals":["trading","trading"]}`,
		"unknown goal":  `{"capacity":350,"goals":["hoarding"]}`,
		"zero capacity": `{"capacity":0,"goals":["collecting"]}`,
	} {
		rec := api.do(http.MethodPut, "/api/v1/setup", body)
		wantStatus(t, rec, http.StatusUnprocessableEntity)
		if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
}
