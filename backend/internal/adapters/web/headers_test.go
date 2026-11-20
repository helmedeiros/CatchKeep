package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/web"
)

func TestHandler_SendsSecurityHeaders(t *testing.T) {
	site := fstest.MapFS{
		"index.html":    {Data: []byte("<html></html>")},
		"assets/app.js": {Data: []byte("x")},
	}
	for _, path := range []string{"/", "/assets/app.js", "/collection"} {
		rec := httptest.NewRecorder()
		web.Handler(site).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
		h := rec.Header()
		csp := h.Get("Content-Security-Policy")
		for _, part := range []string{"default-src 'self'", "script-src 'self'", "img-src 'self' https://raw.githubusercontent.com", "frame-ancestors 'none'", "object-src 'none'"} {
			if !strings.Contains(csp, part) {
				t.Errorf("%s: CSP %q lacks %q", path, csp, part)
			}
		}
		if h.Get("X-Frame-Options") != "DENY" || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: headers = %v", path, h)
		}
	}
}
