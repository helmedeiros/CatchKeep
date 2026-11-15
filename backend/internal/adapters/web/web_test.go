package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/web"
)

var site = fstest.MapFS{
	"index.html":        {Data: []byte("<html>app</html>")},
	"assets/app-123.js": {Data: []byte("console.log(1)")},
	"favicon.svg":       {Data: []byte("<svg/>")},
}

type response struct {
	status int
	header http.Header
	body   string
}

func get(t *testing.T, h http.Handler, path string) response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	return response{status: rec.Code, header: rec.Header(), body: rec.Body.String()}
}

func TestHandler_ServesIndexAtRoot(t *testing.T) {
	resp := get(t, web.Handler(site), "/")

	if resp.status != http.StatusOK || resp.body != "<html>app</html>" {
		t.Fatalf("GET / = %d, want index.html", resp.status)
	}
	if cc := resp.header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("index Cache-Control = %q, want no-cache", cc)
	}
}

func TestHandler_ServesHashedAssetsWithLongCache(t *testing.T) {
	resp := get(t, web.Handler(site), "/assets/app-123.js")

	if resp.status != http.StatusOK || resp.body != "console.log(1)" {
		t.Fatalf("GET asset = %d", resp.status)
	}
	if cc := resp.header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("asset Cache-Control = %q, want immutable", cc)
	}
}

func TestHandler_ServesOtherStaticFiles(t *testing.T) {
	resp := get(t, web.Handler(site), "/favicon.svg")

	if resp.status != http.StatusOK || resp.body != "<svg/>" {
		t.Fatalf("GET favicon = %d", resp.status)
	}
}

func TestHandler_FallsBackToIndexForClientRoutes(t *testing.T) {
	resp := get(t, web.Handler(site), "/collection/abc")

	if resp.status != http.StatusOK || resp.body != "<html>app</html>" {
		t.Fatalf("GET client route = %d, want index.html", resp.status)
	}
}

func TestHandler_MissingAssetIsNotFound(t *testing.T) {
	resp := get(t, web.Handler(site), "/assets/missing.js")

	if resp.status != http.StatusNotFound {
		t.Fatalf("GET missing asset = %d, want 404", resp.status)
	}
}
