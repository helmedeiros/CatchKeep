package sqlite_test

import (
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSignInLog_RecordListAndSweep(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	log := sqlite.NewSignInLog(store)
	ctx := t.Context()
	for _, e := range []domain.SignInEntry{
		{Kind: domain.SignedIn, TrainerID: "ash", Device: "192.168.1.20", At: at.Add(-91 * 24 * time.Hour)},
		{Kind: domain.WrongPIN, Device: "192.168.1.30", At: at.Add(-time.Hour)},
		{Kind: domain.SignInPaused, TrainerID: "ash", Device: "192.168.1.30", At: at},
	} {
		if err := log.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := log.List(ctx, 2)
	if err != nil || len(got) != 2 || got[0].Kind != domain.SignInPaused || got[0].TrainerID != "ash" || got[1].TrainerID != "" || !got[0].At.Equal(at) {
		t.Fatalf("List = %+v, %v; want the newest two, the unknown profile as none", got, err)
	}
	n, err := log.DeleteBefore(ctx, at.Add(-domain.SignInLogLifetime))
	if err != nil || n != 1 {
		t.Fatalf("DeleteBefore = %d, %v, want the 91-day-old entry", n, err)
	}
	if left, _ := log.List(ctx, 10); len(left) != 2 {
		t.Fatalf("left = %+v", left)
	}
}
