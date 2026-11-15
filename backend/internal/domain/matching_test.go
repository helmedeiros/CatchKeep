package domain_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func snap(kv ...string) domain.Snapshot {
	s := domain.Snapshot{}
	for i := 0; i+1 < len(kv); i += 2 {
		s[domain.Field(kv[i])] = kv[i+1]
	}
	return s
}

func base() []string {
	return []string{"species", "Pidgey", "form", "Normal", "gender", "male", "iv_attack", "1", "iv_defense", "2", "iv_stamina", "3", "catch_date", "2026-01-02"}
}

func TestMightBeSameCatch_SameSpeciesFormGenderIVsAndDate(t *testing.T) {
	if !domain.MightBeSameCatch(snap(base()...), snap(append(base(), "cp", "999")...)) {
		t.Fatal("identical identity with different CP (power-up) not flagged")
	}
}

func TestMightBeSameCatch_UnknownCatchDateOrGenderCannotRuleItOut(t *testing.T) {
	noDate := snap("species", "Pidgey", "form", "Normal", "gender", "male", "iv_attack", "1", "iv_defense", "2", "iv_stamina", "3")
	noGender := snap("species", "pidgey", "form", "Normal", "iv_attack", "1", "iv_defense", "2", "iv_stamina", "3", "catch_date", "2026-01-02")

	if !domain.MightBeSameCatch(snap(base()...), noDate) || !domain.MightBeSameCatch(snap(base()...), noGender) {
		t.Fatal("unknown date/gender treated as proof of a different catch")
	}
}

func TestMightBeSameCatch_AnyKnownDifferenceRulesItOut(t *testing.T) {
	for field, other := range map[string]string{"species": "Rattata", "form": "Alola", "gender": "female", "iv_attack": "15", "catch_date": "2025-01-01"} {
		b := snap(base()...)
		b[domain.Field(field)] = other
		if domain.MightBeSameCatch(snap(base()...), b) {
			t.Fatalf("different %s still flagged", field)
		}
	}
}
