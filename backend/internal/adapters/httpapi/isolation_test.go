package httpapi_test

import (
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/csvimport"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/httpapi"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/ids"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/refdata"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
)

// sharedRoutes serve no trainer's data: they are public, the host's own
// settings, or the sign-in flow itself.
var sharedRoutes = []string{
	"GET /health", "GET /host", "GET /autostart", "PUT /autostart",
	"GET /profiles", "POST /profiles", "PUT /profiles/{profileId}/pin",
	"POST /session", "DELETE /session", "GET /me",
}

// trainerReads are the routes that show a trainer their own data.
var trainerReads = []string{
	"/collection", "/setup", "/history", "/plan?kind=limit", "/species-rules",
	"/category-rules", "/teams", "/keep-summary", "/capacity-bands",
}

// signedInToo signs a second profile in on the same server.
func (c *apiClient) signedInToo(name string) *apiClient {
	c.t.Helper()
	other := &apiClient{t: c.t, handler: c.handler}
	created := other.do(http.MethodPost, "/api/v1/profiles", `{"displayName":"`+name+`","pin":"2468"}`)
	p := decode[struct{ ID string }](c.t, created.Body.Bytes())
	wantStatus(c.t, other.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+p.ID+`","pin":"2468"}`), http.StatusOK)
	return other
}

func TestIsolation_NoRouteReachesAnotherProfilesData(t *testing.T) {
	t.Run("in memory", func(t *testing.T) { checkIsolation(t, newAPI(t)) })
	t.Run("in the database", func(t *testing.T) { checkIsolation(t, newSQLiteAPI(t)) })
}

// newSQLiteAPI serves the API from a real database, where the queries that
// keep profiles apart live.
func newSQLiteAPI(t *testing.T) *apiClient {
	t.Helper()
	store, err := sqlite.Open(t.Context(), filepath.Join(t.TempDir(), "catchkeep.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	clock := apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	trainers, collection, teams := sqlite.NewTrainers(store), sqlite.NewCollection(store), sqlite.NewTeams(store)
	return &apiClient{t: t, handler: httpapi.NewHandler(httpapi.Services{
		Health:     app.NewHealth("test"),
		Profiles:   app.NewProfiles(trainers, apptest.PlainHasher{}, ids.NewULID(), clock),
		Sessions:   app.NewSessions(trainers, sqlite.NewSessions(store), apptest.PlainHasher{}, apptest.NewSequentialTokens(), clock),
		Imports:    app.NewImports(csvimport.NewPokeGenie(time.UTC), csvimport.Generic{}, collection, collection, ids.NewULID(), clock),
		Collection: app.NewCollection(collection, collection, ids.NewULID(), clock),
		Keep:       app.NewKeep(sqlite.NewSetups(store), collection, refdata.NewSpecies(), teams),
		Teams:      app.NewTeams(teams, collection, ids.NewULID(), clock),
		HostNames:  func() []string { return []string{hostAddress} },
		Autostart:  app.NewAutostart(&apptest.Autostart{}),
		Transfers:  app.NewTransfers(sqlite.NewTransfers(store), collection, collection, ids.NewULID(), clock),
	})}
}

func checkIsolation(t *testing.T, api *apiClient) {
	ash := api.signedInToo("Ash")
	all := `{"features":["shiny","costume","background","dynamax","gigantamax"]}`
	committed := decode[preview](t, ash.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, ash.do(http.MethodPost, "/api/v1/imports/"+committed.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, ash.do(http.MethodPost, "/api/v1/collection/confirmations", all), http.StatusOK)
	wantStatus(t, ash.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusOK)
	list := decode[struct{ Pokemon []struct{ ID string } }](t, ash.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()).Pokemon
	kept, moved := list[0].ID, list[1].ID
	wantStatus(t, ash.do(http.MethodPut, "/api/v1/collection/"+kept+"/protection", `{"protected":true}`), http.StatusOK)
	wantStatus(t, ash.do(http.MethodPut, "/api/v1/species-rules", `{"rules":[{"species":"Mewtwo","minimum":1,"favorite":false}]}`), http.StatusOK)
	wantStatus(t, ash.do(http.MethodPut, "/api/v1/category-rules", `{"rules":[{"category":"legendary","minimum":1,"favorite":false}]}`), http.StatusOK)
	team := decode[struct{ ID string }](t, ash.do(http.MethodPost, "/api/v1/teams", `{"name":"Raid","memberIds":["`+kept+`"]}`).Body.Bytes())
	event := decode[struct{ ID string }](t, ash.do(http.MethodPost, "/api/v1/transfers", `{"pokemonIds":["`+moved+`"],"requestId":"ash-1"}`).Body.Bytes())
	staged := decode[preview](t, ash.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	row := staged.PossibleMatches[0].Row
	before := snapshot(ash)
	ashIDs := []string{committed.ID, staged.ID, team.ID, event.ID}
	for _, p := range list {
		ashIDs = append(ashIDs, p.ID)
	}

	misty := ash.signedInToo("Misty")
	own := decode[preview](t, misty.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	if own.PossibleMatchCount != 0 {
		t.Fatalf("Misty's preview matched %d Pokémon of Ash's", own.PossibleMatchCount)
	}
	sameAsAsh := `{"answers":[{"row":2,"answer":"same","sameAs":"` + kept + `"}]}`

	cases := []attempt{
		{"GET /imports/{importId}", http.MethodGet, "/imports/" + staged.ID, "", http.StatusNotFound},
		{"POST /imports/{importId}/commit", http.MethodPost, "/imports/" + staged.ID + "/commit", "", http.StatusNotFound},
		{"PUT /imports/{importId}/mapping", http.MethodPut, "/imports/" + staged.ID + "/mapping", `{"mapping":[]}`, http.StatusNotFound},
		{"POST /imports/{importId}/answers", http.MethodPost, "/imports/" + staged.ID + "/answers", `{"answers":[{"row":` + strconv.Itoa(row) + `,"answer":"different"}]}`, http.StatusNotFound},
		{"POST /imports/{importId}/answers", http.MethodPost, "/imports/" + own.ID + "/answers", sameAsAsh, http.StatusUnprocessableEntity},
		{"PUT /collection/{pokemonId}/features", http.MethodPut, "/collection/" + kept + "/features", `{"shiny":true}`, http.StatusNotFound},
		{"PUT /collection/{pokemonId}/protection", http.MethodPut, "/collection/" + kept + "/protection", `{"protected":false}`, http.StatusNotFound},
		{"POST /transfers", http.MethodPost, "/transfers", `{"pokemonIds":["` + kept + `"],"requestId":"ash-1"}`, http.StatusConflict},
		{"POST /transfers/{eventId}/undo", http.MethodPost, "/transfers/" + event.ID + "/undo", "", http.StatusNotFound},
		{"POST /teams", http.MethodPost, "/teams", `{"name":"Mine","memberIds":["` + kept + `"]}`, http.StatusUnprocessableEntity},
		{"PUT /teams/{teamId}", http.MethodPut, "/teams/" + team.ID, `{"name":"Taken","memberIds":[]}`, http.StatusNotFound},
		{"DELETE /teams/{teamId}", http.MethodDelete, "/teams/" + team.ID, "", http.StatusNotFound},
		{"POST /imports/{importId}/commit", http.MethodPost, "/imports/" + own.ID + "/commit", "", http.StatusOK},
		{"POST /collection/confirmations", http.MethodPost, "/collection/confirmations", all, http.StatusOK},
		{"PUT /setup", http.MethodPut, "/setup", `{"capacity":50,"goals":["trading"],"inGameCount":40}`, http.StatusOK},
		{"PUT /species-rules", http.MethodPut, "/species-rules", `{"rules":[]}`, http.StatusOK},
		{"PUT /category-rules", http.MethodPut, "/category-rules", `{"rules":[]}`, http.StatusOK},
	}
	for _, r := range trainerReads {
		route, _, _ := strings.Cut(r, "?")
		cases = append(cases, attempt{"GET " + route, http.MethodGet, r, "", http.StatusOK})
	}

	covered := []string{"POST /imports"}
	for _, c := range cases {
		covered = append(covered, c.route)
		rec := misty.do(c.method, "/api/v1"+c.path, c.body)
		if rec.Code != c.want {
			t.Errorf("%s as Misty: status = %d, want %d; body: %s", c.route, rec.Code, c.want, rec.Body)
		}
		for _, id := range ashIDs {
			if !strings.Contains(c.path+c.body, id) && strings.Contains(rec.Body.String(), id) {
				t.Errorf("%s as Misty shows Ash's %s: %s", c.route, id, rec.Body)
			}
		}
		if made := c.madeUp(ashIDs); rec.Code >= 400 && made != c {
			if again := misty.do(made.method, "/api/v1"+made.path, made.body); again.Code != rec.Code || errorCode(t, again) != errorCode(t, rec) {
				t.Errorf("%s tells Ash's ids from made-up ones: %d %s, then %d %s", c.route, rec.Code, rec.Body, again.Code, again.Body)
			}
		}
	}
	for _, route := range contractRoutes(t) {
		if !slices.Contains(covered, route) && !slices.Contains(sharedRoutes, route) {
			t.Errorf("%s is not checked for isolation; add it to this test or to sharedRoutes", route)
		}
	}
	for path, body := range snapshot(ash) {
		if body != before[path] {
			t.Errorf("Misty changed Ash's %s:\nbefore %s\nafter  %s", path, before[path], body)
		}
	}
}

// attempt is one call Misty makes, and the status it must get.
type attempt struct {
	route, method, path, body string
	want                      int
}

// madeUp is the same attempt with Ash's ids swapped for ones nobody has.
func (a attempt) madeUp(ids []string) attempt {
	for _, id := range ids {
		a.path = strings.ReplaceAll(a.path, id, "01JD2Z8Q4M8V7Y6X5W4T3S2ZZZ")
		a.body = strings.ReplaceAll(a.body, id, "01JD2Z8Q4M8V7Y6X5W4T3S2ZZZ")
	}
	return a
}

// snapshot reads everything a trainer can see, by path.
func snapshot(c *apiClient) map[string]string {
	c.t.Helper()
	seen := map[string]string{}
	for _, path := range trainerReads {
		rec := c.do(http.MethodGet, "/api/v1"+path, "")
		wantStatus(c.t, rec, http.StatusOK)
		seen[path] = rec.Body.String()
	}
	return seen
}

// contractRoutes lists every "METHOD /path" in the contract.
func contractRoutes(t *testing.T) []string {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var routes []string
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			routes = append(routes, method+" "+path)
		}
	}
	return routes
}
