package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/csvimport"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/httpapi"
	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

type apiClient struct {
	t       *testing.T
	handler http.Handler
	cookie  *http.Cookie
}

func newAPI(t *testing.T) *apiClient {
	t.Helper()
	trainers := apptest.NewTrainers()
	clock := apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	hasher := apptest.PlainHasher{}
	collection := apptest.NewCollectionStore()
	teams := apptest.NewTeams(collection)
	return &apiClient{t: t, handler: httpapi.NewHandler(httpapi.Services{
		Health:     app.NewHealth("test"),
		Profiles:   app.NewProfiles(trainers, hasher, ulidLike{}, clock),
		Sessions:   app.NewSessions(trainers, apptest.NewSessionStore(), hasher, apptest.NewSequentialTokens(), clock),
		Imports:    app.NewImports(csvimport.NewPokeGenie(time.UTC), csvimport.Generic{}, collection, collection, ulidLike{}, clock),
		Collection: app.NewCollection(collection, collection, clock),
		Keep:       app.NewKeep(apptest.NewSetups(), collection, apptest.SpeciesCategories{"Mewtwo": domain.CategoryLegendary}, teams),
		Teams:      app.NewTeams(teams, ulidLike{}),
	})}
}

// ulidLike returns contract-shaped ids so responses validate.
type ulidLike struct{}

var ulidSeq = 0

func (ulidLike) NewID() string {
	ulidSeq++
	s := "01JD2Z8Q4M8V7Y6X5W4T3S2R1P"
	n := []byte(s)
	n[25] = "0123456789ABCDEFGHJKMNPQRS"[ulidSeq%26]
	n[24] = "0123456789ABCDEFGHJKMNPQRS"[(ulidSeq/26)%26]
	return string(n)
}

func (c *apiClient) do(method, path, body string) *httptest.ResponseRecorder {
	c.t.Helper()
	return c.doWithType(method, path, body, "application/json")
}

func (c *apiClient) doWithType(method, path, body, contentType string) *httptest.ResponseRecorder {
	c.t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequestWithContext(c.t.Context(), method, path, http.NoBody)
	} else {
		req = httptest.NewRequestWithContext(c.t.Context(), method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	assertMatchesContract(c.t, req, rec)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "ck_session" {
			if ck.MaxAge < 0 {
				c.cookie = nil
			} else {
				c.cookie = ck
			}
		}
	}
	return rec
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, status, rec.Body)
	}
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code
}

func TestProfiles_CreateListSignInMeSignOut(t *testing.T) {
	api := newAPI(t)

	created := api.do(http.MethodPost, "/api/v1/profiles", `{"displayName":"Ash","pin":"2468"}`)
	wantStatus(t, created, http.StatusCreated)
	var ash struct{ ID, DisplayName string }
	_ = json.Unmarshal(created.Body.Bytes(), &ash)

	list := api.do(http.MethodGet, "/api/v1/profiles", "")
	wantStatus(t, list, http.StatusOK)
	if !strings.Contains(list.Body.String(), `"displayName":"Ash"`) {
		t.Fatalf("list = %s", list.Body)
	}

	wantStatus(t, api.do(http.MethodGet, "/api/v1/me", ""), http.StatusUnauthorized)

	signIn := api.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+ash.ID+`","pin":"2468"}`)
	wantStatus(t, signIn, http.StatusOK)
	if api.cookie == nil || !api.cookie.HttpOnly || api.cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie = %+v, want HttpOnly SameSite=Strict", api.cookie)
	}

	me := api.do(http.MethodGet, "/api/v1/me", "")
	wantStatus(t, me, http.StatusOK)
	if !strings.Contains(me.Body.String(), `"displayName":"Ash"`) {
		t.Fatalf("me = %s", me.Body)
	}

	wantStatus(t, api.do(http.MethodDelete, "/api/v1/session", ""), http.StatusNoContent)
	if api.cookie != nil {
		t.Fatal("session cookie not cleared on sign-out")
	}
	wantStatus(t, api.do(http.MethodGet, "/api/v1/me", ""), http.StatusUnauthorized)
}

func TestProfiles_ErrorsUseTheContractEnvelope(t *testing.T) {
	api := newAPI(t)
	created := api.do(http.MethodPost, "/api/v1/profiles", `{"displayName":"Ash","pin":"2468"}`)
	var ash struct{ ID string }
	_ = json.Unmarshal(created.Body.Bytes(), &ash)

	cases := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"short PIN", http.MethodPost, "/api/v1/profiles", `{"displayName":"Misty","pin":"12"}`, http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{"malformed JSON", http.MethodPost, "/api/v1/profiles", `{"displayName":`, http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{"duplicate name", http.MethodPost, "/api/v1/profiles", `{"displayName":"ash","pin":"1357"}`, http.StatusConflict, "DISPLAY_NAME_TAKEN"},
		{"wrong PIN", http.MethodPost, "/api/v1/session", `{"profileId":"` + ash.ID + `","pin":"1111"}`, http.StatusUnauthorized, "PIN_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api.t = t
			rec := api.do(tc.method, tc.path, tc.body)
			wantStatus(t, rec, tc.status)
			if got := errorCode(t, rec); got != tc.code {
				t.Fatalf("code = %q, want %q", got, tc.code)
			}
		})
	}
}

func TestSignIn_RepeatedWrongPINsAreRateLimited(t *testing.T) {
	api := newAPI(t)
	created := api.do(http.MethodPost, "/api/v1/profiles", `{"displayName":"Ash","pin":"2468"}`)
	var ash struct{ ID string }
	_ = json.Unmarshal(created.Body.Bytes(), &ash)
	for range 5 {
		api.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+ash.ID+`","pin":"1111"}`)
	}

	rec := api.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+ash.ID+`","pin":"2468"}`)

	wantStatus(t, rec, http.StatusTooManyRequests)
	if got := errorCode(t, rec); got != "RATE_LIMITED" {
		t.Fatalf("code = %q, want RATE_LIMITED", got)
	}
}
