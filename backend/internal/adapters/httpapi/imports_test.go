package httpapi_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func signedIn(t *testing.T, name string) *apiClient {
	t.Helper()
	api := newAPI(t)
	created := api.do(http.MethodPost, "/api/v1/profiles", `{"displayName":"`+name+`","pin":"2468"}`)
	var p struct{ ID string }
	_ = json.Unmarshal(created.Body.Bytes(), &p)
	wantStatus(t, api.do(http.MethodPost, "/api/v1/session", `{"profileId":"`+p.ID+`","pin":"2468"}`), http.StatusOK)
	return api
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
	PossibleMatchCount int
	InvalidRows        []struct{ Row int }
	UnknownFields      []string
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
	api := signedIn(t, "Ash")

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
	api := signedIn(t, "Ash")
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

	ash := signedIn(t, "Ash")
	p := decode[preview](t, ash.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv").Body.Bytes())
	misty := signedIn(t, "Misty")
	wantStatus(t, misty.do(http.MethodGet, "/api/v1/imports/"+p.ID, ""), http.StatusNotFound)
}

func TestImport_RejectsUnrecognizedAndOversizedFiles(t *testing.T) {
	api := signedIn(t, "Ash")

	rec := api.doWithType(http.MethodPost, "/api/v1/imports", "name,cp\nPikachu,500\n", "text/csv")
	wantStatus(t, rec, http.StatusUnprocessableEntity)
	if code := errorCode(t, rec); code != "UNSUPPORTED_FILE" {
		t.Fatalf("code = %q", code)
	}

	huge := fixture(t) + strings.Repeat("x", 11<<20)
	wantStatus(t, api.doWithType(http.MethodPost, "/api/v1/imports", huge, "text/csv"), http.StatusRequestEntityTooLarge)
}
