package httpapi_test

import (
	"net/http"
	"testing"
)

func TestKeep_SetupThenSummary(t *testing.T) {
	api := signedIn(t)
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
		Total, Kept, Surplus, NeedsConfirmation, Minimum, RareMinimum int
		Groups                                                        []struct {
			Species       string
			Rare          bool
			Minimum, Keep int
			KeptIDs       []string `json:"keptIds"`
			SurplusIDs    []string `json:"surplusIds"`
		}
	}](t, api.do(http.MethodGet, "/api/v1/keep-summary", "").Body.Bytes())
	if summary.Total != 4 || summary.Minimum != 1 || summary.NeedsConfirmation != 0 || summary.Kept+summary.Surplus != 4 || len(summary.Groups) == 0 {
		t.Fatalf("summary = %+v", summary)
	}
	for _, g := range summary.Groups {
		if len(g.KeptIDs) != g.Keep {
			t.Errorf("group %+v: keptIds must list the %d kept copies", g, g.Keep)
		}
		if rare := g.Species == "Mewtwo"; g.Rare != rare || (rare && g.Minimum != summary.RareMinimum) {
			t.Errorf("group %+v: Mewtwo is the rare species here, with minimum %d", g, summary.RareMinimum)
		}
	}
}

func TestKeep_RejectsInvalidSetupAndAnonymousRequests(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/keep-summary", ""), http.StatusUnauthorized)
	wantStatus(t, newAPI(t).do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusUnauthorized)

	api := signedIn(t)
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

func TestKeep_SpeciesRules(t *testing.T) {
	api := signedIn(t)
	rules := `{"rules":[{"species":"Mewtwo","minimum":1,"favorite":false}]}`
	wantStatus(t, api.do(http.MethodGet, "/api/v1/species-rules", ""), http.StatusConflict)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/species-rules", rules), http.StatusConflict)

	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"features":["shiny","costume","background","dynamax","gigantamax"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/species-rules", `{"rules":[{"species":"Mewtwo","minimum":0,"favorite":false}]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/species-rules", rules), http.StatusOK)

	got := decode[struct {
		Rules []struct {
			Species string
			Minimum int
		}
	}](t, api.do(http.MethodGet, "/api/v1/species-rules", "").Body.Bytes())
	if len(got.Rules) != 1 || got.Rules[0].Species != "Mewtwo" || got.Rules[0].Minimum != 1 {
		t.Fatalf("rules = %+v", got)
	}
	summary := decode[struct {
		Groups []struct {
			Species, Rule string
			Default       int
		}
	}](t, api.do(http.MethodGet, "/api/v1/keep-summary", "").Body.Bytes())
	for _, g := range summary.Groups {
		if g.Species == "Mewtwo" && (g.Rule != "species" || g.Default != 2) {
			t.Fatalf("Mewtwo group rule = %q replacing default %d, want species replacing the rare 2", g.Rule, g.Default)
		}
	}
}

func TestKeep_ProtectingAPokemon(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodPut, "/api/v1/collection/X/protection", `{"protected":true}`), http.StatusUnauthorized)
	api := signedIn(t)
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusOK)
	type pokemon struct {
		ID        string
		Protected bool
	}
	list := decode[struct{ Pokemon []pokemon }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	id := list.Pokemon[0].ID

	wantStatus(t, api.do(http.MethodPut, "/api/v1/collection/missing/protection", `{"protected":true}`), http.StatusNotFound)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/collection/"+id+"/protection", `{"protected":true}`), http.StatusOK)

	list = decode[struct{ Pokemon []pokemon }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	if !list.Pokemon[0].Protected {
		t.Fatalf("collection = %+v; the first Pokémon is protected", list)
	}
	summary := decode[struct{ Protected int }](t, api.do(http.MethodGet, "/api/v1/keep-summary", "").Body.Bytes())
	if summary.Protected != 0 {
		t.Fatalf("protected = %d; unconfirmed Pokémon are not in a group yet", summary.Protected)
	}
}

func TestKeep_CapacityBands(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/capacity-bands", ""), http.StatusUnauthorized)

	got := decode[struct {
		Bands []struct{ UpTo, Ordinary, Rare int }
	}](t, signedIn(t).do(http.MethodGet, "/api/v1/capacity-bands", "").Body.Bytes())
	if len(got.Bands) != 5 || got.Bands[0].UpTo != 500 || got.Bands[0].Ordinary != 1 || got.Bands[0].Rare != 2 {
		t.Fatalf("bands = %+v", got.Bands)
	}
}

func TestKeep_CategoryRules(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/category-rules", ""), http.StatusUnauthorized)
	api := signedIn(t)
	rules := `{"rules":[{"category":"legendary","minimum":1,"favorite":false}]}`
	wantStatus(t, api.do(http.MethodGet, "/api/v1/category-rules", ""), http.StatusConflict)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/category-rules", rules), http.StatusConflict)

	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"features":["shiny","costume","background","dynamax","gigantamax"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/category-rules", `{"rules":[{"category":"legendary","minimum":0,"favorite":false}]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/category-rules", rules), http.StatusOK)

	got := decode[struct {
		Rules []struct {
			Category string
			Minimum  int
		}
	}](t, api.do(http.MethodGet, "/api/v1/category-rules", "").Body.Bytes())
	if len(got.Rules) != 1 || got.Rules[0].Category != "legendary" || got.Rules[0].Minimum != 1 {
		t.Fatalf("rules = %+v", got)
	}
	summary := decode[struct {
		Groups []struct{ Species, Rule, Category string }
	}](t, api.do(http.MethodGet, "/api/v1/keep-summary", "").Body.Bytes())
	for _, g := range summary.Groups {
		if g.Species == "Mewtwo" && (g.Rule != "category" || g.Category != "legendary") {
			t.Fatalf("Mewtwo group = %+v, want the legendary category rule", g)
		}
	}
}

func TestCollection_MarksOnePokemonsFeatures(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodPut, "/api/v1/collection/X/features", `{"shiny":true}`), http.StatusUnauthorized)
	api := signedIn(t)
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	type pokemon struct {
		ID         string
		Attributes map[string]string
	}
	id := decode[struct{ Pokemon []pokemon }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()).Pokemon[0].ID

	wantStatus(t, api.do(http.MethodPut, "/api/v1/collection/missing/features", `{"shiny":true}`), http.StatusNotFound)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/collection/"+id+"/features", `{"shadow":true,"purified":true}`), http.StatusUnprocessableEntity)
	rec := api.do(http.MethodPut, "/api/v1/collection/"+id+"/features", `{"purified":true,"shiny":false}`)
	wantStatus(t, rec, http.StatusOK)
	got := decode[pokemon](t, rec.Body.Bytes())
	if got.ID != id || got.Attributes["purified"] != "yes" || got.Attributes["shadow"] != "no" || got.Attributes["shiny"] != "no" {
		t.Fatalf("marked = %+v", got)
	}
}

func TestKeep_PlanSuggestsTransfersForATarget(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/plan?kind=limit", ""), http.StatusUnauthorized)
	api := signedIn(t)
	wantStatus(t, api.do(http.MethodGet, "/api/v1/plan?kind=limit", ""), http.StatusConflict)

	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"features":["shiny","costume","background","dynamax","gigantamax"]}`), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/setup", `{"capacity":350,"goals":["collecting"],"inGameCount":300,"limit":80}`), http.StatusOK)
	setup := decode[struct{ InGameCount, Limit int }](t, api.do(http.MethodGet, "/api/v1/setup", "").Body.Bytes())
	if setup.InGameCount != 300 || setup.Limit != 80 {
		t.Fatalf("setup = %+v", setup)
	}

	type plan struct {
		Use                            struct{ Occupied, Limit int }
		Wanted, MaxFeasible, Shortfall int
		Transfers                      []struct {
			PokemonID string `json:"pokemonId"`
		}
	}
	got := decode[plan](t, api.do(http.MethodGet, "/api/v1/plan?kind=limit", "").Body.Bytes())
	if got.Use.Occupied != 300 || got.Use.Limit != 280 || got.Wanted != 20 || got.Shortfall != got.Wanted-len(got.Transfers) {
		t.Fatalf("plan = %+v", got)
	}
	searches := decode[struct {
		Searches []struct{ Species, Query string }
	}](t, api.do(http.MethodGet, "/api/v1/plan?kind=limit", "").Body.Bytes()).Searches
	if len(got.Transfers) > 0 && (len(searches) == 0 || searches[0].Query == "") {
		t.Fatalf("searches = %+v; every species with a Pokédex number gets one", searches)
	}
	free := decode[plan](t, api.do(http.MethodGet, "/api/v1/plan?kind=free_more&value=1", "").Body.Bytes())
	if free.Wanted != 1 || len(free.Transfers) != min(1, free.MaxFeasible) {
		t.Fatalf("plan = %+v", free)
	}
	wantStatus(t, api.do(http.MethodGet, "/api/v1/plan?kind=free_more", ""), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodGet, "/api/v1/plan?kind=percent&value=500", ""), http.StatusUnprocessableEntity)
}

func TestTransfers_ConfirmUndoAndHistory(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodGet, "/api/v1/history", ""), http.StatusUnauthorized)
	api := signedIn(t)
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	type pokemon struct{ ID string }
	list := decode[struct{ Pokemon []pokemon }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()).Pokemon
	id := list[0].ID
	type event struct {
		ID, Kind, Undoes, UndoneBy string
		Pokemon                    []struct{ ID, Species string }
	}

	wantStatus(t, api.do(http.MethodPost, "/api/v1/transfers", `{"pokemonIds":["`+id+`"],"requestId":"r1","planRevision":999}`), http.StatusConflict)
	rec := api.do(http.MethodPost, "/api/v1/transfers", `{"pokemonIds":["`+id+`"],"requestId":"r1"}`)
	wantStatus(t, rec, http.StatusCreated)
	e := decode[event](t, rec.Body.Bytes())
	if e.Kind != "single" || len(e.Pokemon) != 1 || e.Pokemon[0].ID != id {
		t.Fatalf("event = %+v", e)
	}
	if again := decode[event](t, api.do(http.MethodPost, "/api/v1/transfers", `{"pokemonIds":["`+id+`"],"requestId":"r1"}`).Body.Bytes()); again.ID != e.ID {
		t.Fatalf("repeat = %+v", again)
	}
	rec = api.do(http.MethodPost, "/api/v1/transfers", `{"pokemonIds":["`+id+`"],"requestId":"r2"}`)
	wantStatus(t, rec, http.StatusConflict)
	if code := errorCode(t, rec); code != "ALREADY_TRANSFERRED" {
		t.Fatalf("code = %q", code)
	}
	if left := decode[struct{ Pokemon []pokemon }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()).Pokemon; len(left) != len(list)-1 {
		t.Fatalf("collection = %d, want %d", len(left), len(list)-1)
	}

	wantStatus(t, api.do(http.MethodPost, "/api/v1/transfers/nope/undo", ""), http.StatusNotFound)
	undo := decode[event](t, api.do(http.MethodPost, "/api/v1/transfers/"+e.ID+"/undo", "").Body.Bytes())
	if undo.Kind != "undo" || undo.Undoes != e.ID {
		t.Fatalf("undo = %+v", undo)
	}
	wantStatus(t, api.do(http.MethodPost, "/api/v1/transfers/"+e.ID+"/undo", ""), http.StatusConflict)
	history := decode[struct{ Events []event }](t, api.do(http.MethodGet, "/api/v1/history", "").Body.Bytes()).Events
	if len(history) != 2 || history[0].Kind != "undo" || history[1].UndoneBy != undo.ID {
		t.Fatalf("history = %+v", history)
	}
}
