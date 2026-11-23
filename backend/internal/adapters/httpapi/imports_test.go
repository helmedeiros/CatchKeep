package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

// signedIn is a new server with Ash signed in.
func signedIn(t *testing.T) *apiClient {
	t.Helper()
	return newAPI(t).signedInToo("Ash")
}

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../csvimport/testdata/poke_genie_sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type preview struct {
	ID                 string
	Status             string
	TotalRows          int
	NewCount           int
	UpdatedCount       int
	PossibleMatchCount int
	PossibleMatches    []struct {
		Row        int
		Exact      bool
		Answer     string
		Incoming   struct{ ID, Species string }
		Candidates []struct {
			Origin  string
			Pokemon struct{ ID, Species string }
		}
	}
	InvalidRows   []struct{ Row int }
	UnknownFields []string
}

func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func TestImport_StagePreviewCommitAndBrowse(t *testing.T) {
	api := signedIn(t)

	staged := api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv")
	wantStatus(t, staged, http.StatusCreated)
	p := decode[preview](t, staged.Body.Bytes())
	if p.Status != "staged" || p.TotalRows != 5 || p.NewCount != 4 || len(p.InvalidRows) != 2 || len(p.UnknownFields) == 0 {
		t.Fatalf("preview = %+v", p)
	}

	wantStatus(t, api.do(http.MethodGet, "/api/v1/collection", ""), http.StatusOK)
	if c := decode[struct{ Total int }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()); c.Total != 0 {
		t.Fatalf("collection has %d before commit", c.Total)
	}

	wantStatus(t, api.do(http.MethodGet, "/api/v1/imports/"+p.ID, ""), http.StatusOK)
	committed := api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", "")
	wantStatus(t, committed, http.StatusOK)
	if got := decode[preview](t, committed.Body.Bytes()); got.Status != "committed" || got.NewCount != 4 {
		t.Fatalf("committed = %+v", got)
	}

	coll := decode[struct {
		Total   int
		Pokemon []struct {
			Species    string
			Cp         int
			Attributes map[string]string
		}
	}](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	if coll.Total != 4 || coll.Pokemon[0].Species != "Mewtwo" {
		t.Fatalf("collection = %+v", coll)
	}
	for _, p := range coll.Pokemon {
		if p.Species == "Charizard" && (p.Attributes["lucky"] != "yes" || p.Attributes["shiny"] != "unknown") {
			t.Fatalf("charizard attributes = %v", p.Attributes)
		}
	}
}

func TestImport_ReimportHoldsBackPossibleDuplicates(t *testing.T) {
	api := signedIn(t)
	first := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+first.ID+"/commit", ""), http.StatusOK)

	again := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())

	if again.NewCount != 0 || again.PossibleMatchCount != 4 {
		t.Fatalf("re-import = %+v, want 0 new and 4 possible matches", again)
	}
	rec := api.do(http.MethodPost, "/api/v1/imports/"+first.ID+"/commit", "")
	wantStatus(t, rec, http.StatusConflict)
}

func TestImport_RequiresSignInAndStaysWithinTheProfile(t *testing.T) {
	api := newAPI(t)
	wantStatus(t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv"), http.StatusUnauthorized)
	wantStatus(t, api.do(http.MethodGet, "/api/v1/collection", ""), http.StatusUnauthorized)

	ash := signedIn(t)
	p := decode[preview](t, ash.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	misty := ash.signedInToo("Misty")
	wantStatus(t, misty.do(http.MethodGet, "/api/v1/imports/"+p.ID, ""), http.StatusNotFound)
}

func TestImport_RejectsUnrecognizedAndOversizedFiles(t *testing.T) {
	api := signedIn(t)

	rec := api.doWithType(http.MethodPost, "/api/v1/imports", "just some notes\nnothing tabular\n", "text/csv")
	wantStatus(t, rec, http.StatusUnprocessableEntity)
	if code := errorCode(t, rec); code != "UNSUPPORTED_FILE" {
		t.Fatalf("code = %q", code)
	}

	huge := fixture(t) + strings.Repeat("x", 11<<20)
	wantStatus(t, api.doWithType(http.MethodPost, "/api/v1/imports", huge, "text/csv"), http.StatusRequestEntityTooLarge)
}

func TestImport_AnswersDecideWhetherHeldBackRowsUpdateOrAdd(t *testing.T) {
	api := signedIn(t)
	first := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+first.ID+"/commit", ""), http.StatusOK)
	again := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	if len(again.PossibleMatches) != 4 {
		t.Fatalf("possible matches = %+v", again.PossibleMatches)
	}
	m0, m1 := again.PossibleMatches[0], again.PossibleMatches[1]
	if m0.Answer != "pending" || m0.Incoming.ID != fmt.Sprintf("row-%d", m0.Row) || m0.Candidates[0].Origin != "collection" {
		t.Fatalf("first match = %+v", m0)
	}

	body := fmt.Sprintf(`{"answers":[{"row":%d,"answer":"same","sameAs":%q},{"row":%d,"answer":"different"}]}`,
		m0.Row, m0.Candidates[0].Pokemon.ID, m1.Row)
	answered := api.do(http.MethodPost, "/api/v1/imports/"+again.ID+"/answers", body)
	wantStatus(t, answered, http.StatusOK)
	if got := decode[preview](t, answered.Body.Bytes()); got.UpdatedCount != 1 || got.NewCount != 1 || got.PossibleMatchCount != 2 {
		t.Fatalf("answered = %+v", got)
	}

	committed := decode[preview](t, api.do(http.MethodPost, "/api/v1/imports/"+again.ID+"/commit", "").Body.Bytes())
	if committed.UpdatedCount != 1 || committed.NewCount != 1 || len(committed.PossibleMatches) != 0 {
		t.Fatalf("committed = %+v", committed)
	}
	if c := decode[struct{ Total int }](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes()); c.Total != 5 {
		t.Fatalf("collection has %d, want 4 + 1 different", c.Total)
	}
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+again.ID+"/answers", body), http.StatusConflict)
}

func TestImport_InvalidAnswersAreRejected(t *testing.T) {
	api := signedIn(t)
	first := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+first.ID+"/commit", ""), http.StatusOK)
	again := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	row := again.PossibleMatches[0].Row

	for name, body := range map[string]string{
		"same without a candidate": fmt.Sprintf(`{"answers":[{"row":%d,"answer":"same"}]}`, row),
		"unknown candidate":        fmt.Sprintf(`{"answers":[{"row":%d,"answer":"same","sameAs":"nope"}]}`, row),
		"not a held-back row":      `{"answers":[{"row":999,"answer":"different"}]}`,
	} {
		rec := api.do(http.MethodPost, "/api/v1/imports/"+again.ID+"/answers", body)
		wantStatus(t, rec, http.StatusUnprocessableEntity)
		if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/unknown/answers", `{"answers":[{"row":2,"answer":"different"}]}`), http.StatusNotFound)
}

func TestCollection_ConfirmAbsentForOneSpecies(t *testing.T) {
	api := signedIn(t)
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)

	rec := api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"species":"Charizard","form":"Normal","features":["shiny","costume"]}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[struct{ Updated int }](t, rec.Body.Bytes()); got.Updated != 1 {
		t.Fatalf("updated = %d, want 1", got.Updated)
	}

	coll := decode[struct {
		Pokemon []struct {
			Species    string
			Attributes map[string]string
		}
	}](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	for _, p := range coll.Pokemon {
		want := "unknown"
		if p.Species == "Charizard" {
			want = "no"
		}
		if p.Attributes["shiny"] != want || p.Attributes["costume"] != want || p.Attributes["background"] != "unknown" {
			t.Fatalf("%s attributes = %v", p.Species, p.Attributes)
		}
	}
}

func TestCollection_ConfirmAbsentRejectsInvalidRequests(t *testing.T) {
	wantStatus(t, newAPI(t).do(http.MethodPost, "/api/v1/collection/confirmations", `{"species":"Pidgey","form":"Normal","features":["shiny"]}`), http.StatusUnauthorized)

	api := signedIn(t)
	for name, body := range map[string]string{
		"no features":     `{"species":"Pidgey","form":"Normal","features":[]}`,
		"unknown feature": `{"species":"Pidgey","form":"Normal","features":["cp"]}`,
		"form only":       `{"form":"Normal","features":["shiny"]}`,
		"blank species":   `{"species":" ","form":"Normal","features":["shiny"]}`,
	} {
		rec := api.do(http.MethodPost, "/api/v1/collection/confirmations", body)
		wantStatus(t, rec, http.StatusUnprocessableEntity)
		if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
}

func TestCollection_ConfirmAbsentForTheWholeCollection(t *testing.T) {
	api := signedIn(t)
	p := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)

	rec := api.do(http.MethodPost, "/api/v1/collection/confirmations", `{"features":["background"]}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[struct{ Updated int }](t, rec.Body.Bytes()); got.Updated != 4 {
		t.Fatalf("updated = %d, want all 4", got.Updated)
	}
}

func TestImport_MapsTheColumnsOfAnUnfamiliarFile(t *testing.T) {
	api := signedIn(t)
	type mapped struct {
		ID           string
		Source       string
		NeedsMapping bool
		NewCount     int
		Columns      []struct {
			Name    string
			Samples []string
		}
		Mapping []struct{ Column, Field string }
	}
	rec := api.doWithType(http.MethodPost, "/api/v1/imports", "Mon,Combat Power,Note\nPikachu,500,buddy\n", "text/csv")
	wantStatus(t, rec, http.StatusCreated)
	p := decode[mapped](t, rec.Body.Bytes())
	if !p.NeedsMapping || p.Source != "csv" || len(p.Columns) != 3 || p.Columns[0].Samples[0] != "Pikachu" ||
		len(p.Mapping) != 1 || p.Mapping[0].Column != "Combat Power" || p.Mapping[0].Field != "cp" {
		t.Fatalf("preview = %+v; columns with samples and a suggestion", p)
	}
	rec = api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", "")
	wantStatus(t, rec, http.StatusConflict)
	if code := errorCode(t, rec); code != "IMPORT_NEEDS_MAPPING" {
		t.Fatalf("code = %q", code)
	}

	wantStatus(t, api.do(http.MethodPut, "/api/v1/imports/"+p.ID+"/mapping", `{"mapping":[{"column":"Combat Power","field":"cp"}]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/imports/"+p.ID+"/mapping", `{"mapping":[{"column":"Mon","field":"species"},{"column":"Mon","field":"form"}]}`), http.StatusUnprocessableEntity)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/imports/nope/mapping", `{"mapping":[]}`), http.StatusNotFound)
	rec = api.do(http.MethodPut, "/api/v1/imports/"+p.ID+"/mapping", `{"mapping":[{"column":"Mon","field":"species"},{"column":"Combat Power","field":"cp"}]}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[mapped](t, rec.Body.Bytes()); got.NeedsMapping || got.NewCount != 1 || len(got.Mapping) != 2 {
		t.Fatalf("mapped preview = %+v", got)
	}
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+p.ID+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodPut, "/api/v1/imports/"+p.ID+"/mapping", `{"mapping":[{"column":"Mon","field":"species"}]}`), http.StatusConflict)

	next := decode[mapped](t, api.doWithType(http.MethodPost, "/api/v1/imports", "Mon,Combat Power,Note\nRaichu,900,\n", "text/csv").Body.Bytes())
	if next.NeedsMapping || next.NewCount != 1 {
		t.Fatalf("next preview = %+v; the saved mapping applies", next)
	}
}

func TestImport_ListsCollectedPokemonMissingFromTheFile(t *testing.T) {
	api := signedIn(t)
	first := decode[preview](t, api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+first.ID+"/commit", ""), http.StatusOK)
	var without []string
	for _, line := range strings.Split(fixture(t), "\n") {
		if !strings.Contains(line, "Mewtwo") {
			without = append(without, line)
		}
	}

	type missing struct{ Missing []string }
	staged := api.doWithType(http.MethodPost, "/api/v1/imports", strings.Join(without, "\n"), "text/csv")
	wantStatus(t, staged, http.StatusCreated)
	got := decode[missing](t, staged.Body.Bytes())
	coll := decode[struct {
		Pokemon []struct{ ID, Species string }
	}](t, api.do(http.MethodGet, "/api/v1/collection", "").Body.Bytes())
	if len(got.Missing) != 1 || coll.Pokemon[0].Species != "Mewtwo" || got.Missing[0] != coll.Pokemon[0].ID {
		t.Fatalf("missing = %v, want the Mewtwo of %+v", got.Missing, coll.Pokemon)
	}
}
