package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/httpapi"
	"github.com/helmedeiros/catchkeep/backend/internal/app"
)

func TestGetHealth_ReportsOKAndVersion(t *testing.T) {
	handler := httpapi.NewHandler(httpapi.Services{Health: app.NewHealth("1.2.3")})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/health", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" || body.Version != "1.2.3" {
		t.Fatalf("body = %+v, want status ok and version 1.2.3", body)
	}
	assertMatchesContract(t, req, rec)
}

func TestUnknownRoute_ReturnsNotFound(t *testing.T) {
	handler := httpapi.NewHandler(httpapi.Services{Health: app.NewHealth("1.2.3")})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/nope", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
