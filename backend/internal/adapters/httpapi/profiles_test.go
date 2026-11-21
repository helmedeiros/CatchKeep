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
	// remote is the caller's address and host the name it addresses the
	// server by: the computer running CatchKeep itself unless onOtherDevice.
	remote, host string
}

// hostAddress is the test server's own address on the network.
const hostAddress = "192.168.1.5"

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
		HostNames:  func() []string { return []string{hostAddress} },
		Autostart:  app.NewAutostart(&apptest.Autostart{}),
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
	req.RemoteAddr, req.Host = "127.0.0.1:50000", "localhost:8080"
	if c.remote != "" {
		req.RemoteAddr, req.Host = c.remote, c.host
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

// onOtherDevice makes the client call from another device on the network.
func (c *apiClient) onOtherDevice() *apiClient {
	c.remote, c.host = "192.168.1.20:50000", hostAddress+":8080"
	return c
}

func TestHost_OnlyTheHostIsAdmin(t *testing.T) {
	type host struct{ Admin bool }
	if got := decode[host](t, newAPI(t).do(http.MethodGet, "/api/v1/host", "").Body.Bytes()); !got.Admin {
		t.Fatal("the host itself should be admin")
	}
	if got := decode[host](t, newAPI(t).onOtherDevice().do(http.MethodGet, "/api/v1/host", "").Body.Bytes()); got.Admin {
		t.Fatal("another device on the network should not be admin")
	}
}

func TestHost_TellsOnlyTheHostTheAddressesForOtherDevices(t *testing.T) {
	type host struct{ Addresses []string }
	withAddresses := func() *apiClient {
		return &apiClient{t: t, handler: httpapi.NewHandler(httpapi.Services{
			Addresses: func() []string { return []string{"http://192.168.1.20:8080"} },
			HostNames: func() []string { return []string{hostAddress} },
		})}
	}
	if got := decode[host](t, withAddresses().do(http.MethodGet, "/api/v1/host", "").Body.Bytes()); len(got.Addresses) != 1 {
		t.Fatalf("host sees %v", got.Addresses)
	}
	if got := decode[host](t, withAddresses().onOtherDevice().do(http.MethodGet, "/api/v1/host", "").Body.Bytes()); len(got.Addresses) != 0 {
		t.Fatalf("another device sees %v", got.Addresses)
	}
}

func TestProfiles_OnlyTheHostCreatesThem(t *testing.T) {
	rec := newAPI(t).onOtherDevice().do(http.MethodPost, "/api/v1/profiles", `{"displayName":"Ash","pin":"2468"}`)
	wantStatus(t, rec, http.StatusForbidden)
	if code := errorCode(t, rec); code != "ADMIN_ONLY" {
		t.Fatalf("code = %q", code)
	}
}

func TestProfiles_TheHostResetsAPIN(t *testing.T) {
	device := signedIn(t, "Ash")
	id := decode[[]struct{ ID string }](t, device.do(http.MethodGet, "/api/v1/profiles", "").Body.Bytes())[0].ID
	device.onOtherDevice()
	wantStatus(t, device.do(http.MethodPut, "/api/v1/profiles/"+id+"/pin", `{"pin":"1357"}`), http.StatusForbidden)

	host := &apiClient{t: t, handler: device.handler}
	wantStatus(t, host.do(http.MethodPut, "/api/v1/profiles/"+id+"/pin", `{"pin":"12"}`), http.StatusUnprocessableEntity)
	wantStatus(t, host.do(http.MethodPut, "/api/v1/profiles/01JD2Z8Q4M8V7Y6X5W4T3S2R00/pin", `{"pin":"1357"}`), http.StatusNotFound)
	wantStatus(t, host.do(http.MethodPut, "/api/v1/profiles/"+id+"/pin", `{"pin":"1357"}`), http.StatusNoContent)

	wantStatus(t, device.do(http.MethodGet, "/api/v1/me", ""), http.StatusUnauthorized)
	wantStatus(t, device.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+id+`","pin":"1357"}`), http.StatusOK)
}

func TestAutostart_OnlyTheHostTurnsItOnAndOff(t *testing.T) {
	type state struct{ Supported, Enabled bool }
	api := newAPI(t)
	if got := decode[state](t, api.do(http.MethodGet, "/api/v1/autostart", "").Body.Bytes()); !got.Supported || got.Enabled {
		t.Fatalf("state = %+v", got)
	}
	if got := decode[state](t, api.do(http.MethodPut, "/api/v1/autostart", `{"enabled":true}`).Body.Bytes()); !got.Enabled {
		t.Fatalf("after turning on: %+v", got)
	}
	device := &apiClient{t: t, handler: api.handler}
	device.onOtherDevice()
	wantStatus(t, device.do(http.MethodGet, "/api/v1/autostart", ""), http.StatusForbidden)
	wantStatus(t, device.do(http.MethodPut, "/api/v1/autostart", `{"enabled":false}`), http.StatusForbidden)

	unsupported := &apiClient{t: t, handler: httpapi.NewHandler(httpapi.Services{Autostart: app.NewAutostart(nil)})}
	if got := decode[state](t, unsupported.do(http.MethodGet, "/api/v1/autostart", "").Body.Bytes()); got.Supported {
		t.Fatalf("state = %+v", got)
	}
	wantStatus(t, unsupported.do(http.MethodPut, "/api/v1/autostart", `{"enabled":true}`), http.StatusConflict)
}
