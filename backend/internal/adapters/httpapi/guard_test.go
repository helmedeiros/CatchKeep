package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// raw sends one request as the computer running CatchKeep, then lets the
// caller change it before it is served.
func raw(t *testing.T, api *apiClient, method, path, body string, change func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.RemoteAddr, req.Host = "127.0.0.1:50000", "localhost:8080"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	change(req)
	rec := httptest.NewRecorder()
	api.handler.ServeHTTP(rec, req)
	return rec
}

func TestGuard_RefusesRequestsAddressedToOtherNames(t *testing.T) {
	api := newAPI(t)
	for _, host := range []string{"attacker.example", "attacker.example:8080", "192.168.1.99:8080"} {
		rec := raw(t, api, http.MethodGet, "/api/v1/health", "", func(r *http.Request) { r.Host = host })
		wantStatus(t, rec, http.StatusMisdirectedRequest)
		if code := errorCode(t, rec); code != "HOST_NOT_ALLOWED" {
			t.Errorf("%s: code = %q", host, code)
		}
	}
	for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", hostAddress + ":8080", "LOCALHOST"} {
		wantStatus(t, raw(t, api, http.MethodGet, "/api/v1/health", "", func(r *http.Request) { r.Host = host }), http.StatusOK)
	}
}

func TestGuard_ADNSRebindingPageCannotResetAPIN(t *testing.T) {
	api := signedIn(t)
	id := decode[[]struct{ ID string }](t, api.do(http.MethodGet, "/api/v1/profiles", "").Body.Bytes())[0].ID
	rec := raw(t, api, http.MethodPut, "/api/v1/profiles/"+id+"/pin", `{"pin":"0000"}`, func(r *http.Request) { r.Host = "attacker.example:8080" })
	wantStatus(t, rec, http.StatusMisdirectedRequest)
}

func TestGuard_AdminNeedsThisComputerByName(t *testing.T) {
	api := newAPI(t)
	rec := raw(t, api, http.MethodPost, "/api/v1/profiles", `{"displayName":"Ash","pin":"2468"}`, func(r *http.Request) { r.Host = hostAddress + ":8080" })
	wantStatus(t, rec, http.StatusForbidden)
}

func TestGuard_RefusesCrossSiteChanges(t *testing.T) {
	api := newAPI(t)
	profile := `{"displayName":"Mallory","pin":"1111"}`
	for name, change := range map[string]func(*http.Request){
		"form-style body":        func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") },
		"no content type":        func(r *http.Request) { r.Header.Del("Content-Type") },
		"CSV outside imports":    func(r *http.Request) { r.Header.Set("Content-Type", "text/csv") },
		"another site":           func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		"a sibling site":         func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-site") },
		"foreign origin":         func(r *http.Request) { r.Header.Set("Origin", "http://attacker.example") },
		"origin on another port": func(r *http.Request) { r.Header.Set("Origin", "http://localhost:9999") },
	} {
		rec := raw(t, api, http.MethodPost, "/api/v1/profiles", profile, change)
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("%s: status = %d, want 403 or 415", name, rec.Code)
		}
	}
	if got := decode[[]struct{ DisplayName string }](t, api.do(http.MethodGet, "/api/v1/profiles", "").Body.Bytes()); len(got) != 0 {
		t.Fatalf("profiles = %+v; none should have been created", got)
	}

	ok := raw(t, api, http.MethodPost, "/api/v1/profiles", profile, func(r *http.Request) {
		r.Header.Set("Origin", "http://localhost:8080")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
	})
	wantStatus(t, ok, http.StatusCreated)
}

func TestGuard_AllowsBodylessChangesAndCSVImportsFromTheApp(t *testing.T) {
	api := signedIn(t)
	rec := api.doWithType(http.MethodPost, "/api/v1/imports", fixture(t), "text/csv")
	wantStatus(t, rec, http.StatusCreated)
	id := decode[preview](t, rec.Body.Bytes()).ID
	wantStatus(t, api.do(http.MethodPost, "/api/v1/imports/"+id+"/commit", ""), http.StatusOK)
	wantStatus(t, api.do(http.MethodDelete, "/api/v1/session", ""), http.StatusNoContent)
}

func TestGuard_SendsSecurityHeaders(t *testing.T) {
	rec := newAPI(t).do(http.MethodGet, "/api/v1/health", "")
	for header, want := range map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestGuard_CapsRequestBodies(t *testing.T) {
	api := newAPI(t)
	big := `{"displayName":"` + strings.Repeat("A", 2<<20) + `","pin":"2468"}`
	rec := raw(t, api, http.MethodPost, "/api/v1/profiles", big, func(*http.Request) {})
	wantStatus(t, rec, http.StatusRequestEntityTooLarge)
	if code := errorCode(t, rec); code != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("code = %q", code)
	}
}
