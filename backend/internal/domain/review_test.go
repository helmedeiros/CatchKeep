package domain_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func row(line int, kv ...string) domain.ImportRow {
	var obs []domain.Observation
	for i := 0; i+1 < len(kv); i += 2 {
		obs = append(obs, domain.Observation{Field: domain.Field(kv[i]), Value: kv[i+1], Source: domain.SourceCSV})
	}
	return domain.ImportRow{Line: line, Observations: obs}
}

func mon(id string, kv ...string) domain.Pokemon {
	return domain.Pokemon{ID: domain.PokemonID(id), Observations: row(0, kv...).Observations}
}

func withCP(cp string) []string { return append(base(), "cp", cp) }

func TestReviewImport_NewRowsAreAddedAndPossibleMatchesListTheirCandidates(t *testing.T) {
	existing := []domain.Pokemon{mon("P1", withCP("100")...)}
	rows := []domain.ImportRow{row(2, withCP("150")...), row(3, "species", "Rattata", "cp", "20")}

	r := domain.ReviewImport(existing, rows, nil)

	if len(r.Add) != 1 || r.Add[0].Line != 3 {
		t.Fatalf("Add = %+v, want only line 3", r.Add)
	}
	if len(r.Matches) != 1 || r.Matches[0].Row.Line != 2 || len(r.Matches[0].Candidates) != 1 {
		t.Fatalf("Matches = %+v", r.Matches)
	}
	c := r.Matches[0].Candidates[0]
	if c.Ref != "P1" || c.Pokemon.ID != "P1" {
		t.Fatalf("candidate = %+v, want collection Pokémon P1", c)
	}
	if !r.Matches[0].Exact || r.Pending() != 1 {
		t.Fatalf("match exact=%v pending=%d, want exact and pending", r.Matches[0].Exact, r.Pending())
	}
}

func TestReviewImport_RepeatedRowsInOneFileMatchTheEarlierRow(t *testing.T) {
	r := domain.ReviewImport(nil, []domain.ImportRow{row(2, base()...), row(3, base()...)}, nil)

	if len(r.Add) != 1 || len(r.Matches) != 1 {
		t.Fatalf("review = %+v", r)
	}
	if c := r.Matches[0].Candidates[0]; c.Ref != domain.RowRef(2) || c.Line != 2 {
		t.Fatalf("candidate = %+v, want earlier row 2", c)
	}
}

func TestReviewImport_MatchIsExactOnlyWithOneCandidateAndEveryIdentityFieldKnown(t *testing.T) {
	noDate := []string{"species", "Pidgey", "form", "Normal", "gender", "male", "iv_attack", "1", "iv_defense", "2", "iv_stamina", "3"}
	twoCandidates := []domain.Pokemon{mon("P1", base()...), mon("P2", base()...)}

	if r := domain.ReviewImport([]domain.Pokemon{mon("P1", base()...)}, []domain.ImportRow{row(2, noDate...)}, nil); r.Matches[0].Exact {
		t.Fatal("match with unknown catch date marked exact")
	}
	if r := domain.ReviewImport(twoCandidates, []domain.ImportRow{row(2, base()...)}, nil); r.Matches[0].Exact {
		t.Fatal("match with two candidates marked exact")
	}
}

func TestReviewImport_AppliesTheTrainersAnswers(t *testing.T) {
	existing := []domain.Pokemon{mon("P1", withCP("100")...)}
	rows := []domain.ImportRow{row(2, withCP("150")...)}

	same := domain.ReviewImport(existing, rows, map[int]domain.Resolution{2: {Line: 2, SameAs: "P1"}})
	if len(same.Add) != 0 || len(same.Update) != 1 || same.Update[0].PokemonID != "P1" || same.Pending() != 0 {
		t.Fatalf("same: %+v", same)
	}
	if len(same.Update[0].Observations) != len(rows[0].Observations) {
		t.Fatalf("same: update carries %d observations, want the row's", len(same.Update[0].Observations))
	}

	different := domain.ReviewImport(existing, rows, map[int]domain.Resolution{2: {Line: 2}})
	if len(different.Add) != 1 || len(different.Update) != 0 || different.Pending() != 0 {
		t.Fatalf("different: %+v", different)
	}
}

func TestReviewImport_SameAsAnEarlierRowDropsTheRepeat(t *testing.T) {
	r := domain.ReviewImport(nil, []domain.ImportRow{row(2, base()...), row(3, base()...)},
		map[int]domain.Resolution{3: {Line: 3, SameAs: domain.RowRef(2)}})

	if len(r.Add) != 1 || len(r.Update) != 0 || r.Pending() != 0 || r.Repeats != 1 {
		t.Fatalf("review = %+v", r)
	}
}

func TestReviewImport_AnswerForACandidateThatNoLongerMatchesStaysPending(t *testing.T) {
	r := domain.ReviewImport([]domain.Pokemon{mon("P1", base()...)}, []domain.ImportRow{row(2, base()...)},
		map[int]domain.Resolution{2: {Line: 2, SameAs: "GONE"}})

	if r.Pending() != 1 || len(r.Update) != 0 || len(r.Add) != 0 {
		t.Fatalf("review = %+v", r)
	}
}

func TestReviewImport_ValidateRejectsAnswersThatDoNotFitTheReview(t *testing.T) {
	r := domain.ReviewImport([]domain.Pokemon{mon("P1", base()...)}, []domain.ImportRow{row(2, base()...), row(3, "species", "Rattata")}, nil)

	for name, res := range map[string]domain.Resolution{
		"not a possible match": {Line: 3},
		"unknown line":         {Line: 99},
		"unknown candidate":    {Line: 2, SameAs: "P9"},
	} {
		if err := r.Validate(res); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := r.Validate(domain.Resolution{Line: 2, SameAs: "P1"}); err != nil {
		t.Fatalf("valid answer rejected: %v", err)
	}
}

func TestReviewImport_RowAnsweredDifferentIsACandidateForLaterRows(t *testing.T) {
	existing := []domain.Pokemon{mon("P1", base()...)}
	rows := []domain.ImportRow{row(2, base()...), row(3, base()...)}

	r := domain.ReviewImport(existing, rows, map[int]domain.Resolution{2: {Line: 2}})

	if len(r.Matches) != 2 || len(r.Matches[1].Candidates) != 2 || r.Matches[1].Candidates[1].Ref != domain.RowRef(2) {
		t.Fatalf("later row candidates = %+v", r.Matches[1].Candidates)
	}
}
