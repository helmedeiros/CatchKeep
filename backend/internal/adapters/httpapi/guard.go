package httpapi

import (
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// maxJSONBytes bounds JSON request bodies; the largest real one, answers
// for every row of a big re-import, stays well under it (spec SEC-09).
const maxJSONBytes = 1 << 20

// secureHeaders tells browsers not to sniff, frame, cache or refer API
// responses.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// limitBody caps request bodies: uploads at their own limit, everything
// else at maxJSONBytes.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxJSONBytes)
		if r.URL.Path == BaseURL+"/imports" {
			limit = maxImportBytes + 1
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

func tooLarge(err error) bool {
	var limit *http.MaxBytesError
	return errors.As(err, &limit)
}

// loopbackNames are the names of the computer running CatchKeep as seen
// from itself.
var loopbackNames = []string{"localhost", "127.0.0.1", "::1"}

// hostName returns the host part of a Host header or URL host, lowercased.
func hostName(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		hostport = h
	}
	return strings.ToLower(strings.Trim(hostport, "[]"))
}

// checkHost refuses requests addressed to any name other than this
// computer's, so a page whose domain is pointed at this computer (DNS
// rebinding) cannot use the API.
func checkHost(names func() []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := hostName(r.Host)
			allowed := slices.Contains(loopbackNames, host)
			if !allowed && names != nil {
				allowed = slices.ContainsFunc(names(), func(n string) bool { return strings.EqualFold(n, host) })
			}
			if !allowed {
				writeError(w, http.StatusMisdirectedRequest, errorBody(codeHostNotAllowed,
					"Open CatchKeep by the address shown on the computer running it."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// blockCrossSite refuses changes that did not come from CatchKeep's own
// page (spec SEC-08): another site's form or script must not create
// profiles, reset PINs or change a collection. Bodies must be JSON, or CSV
// for uploads, which a plain form cannot send.
func blockCrossSite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			writeError(w, http.StatusForbidden, errorBody(codeCrossSite, "Changes must come from CatchKeep itself."))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if u, err := url.Parse(origin); err != nil || !strings.EqualFold(u.Host, r.Host) {
				writeError(w, http.StatusForbidden, errorBody(codeCrossSite, "Changes must come from CatchKeep itself."))
				return
			}
		}
		if r.ContentLength != 0 && !allowedBody(r) {
			writeError(w, http.StatusUnsupportedMediaType, errorBody(codeUnsupportedMediaType, "Send JSON, or a CSV file when importing."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedBody(r *http.Request) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return media == "application/json" || (media == "text/csv" && r.URL.Path == BaseURL+"/imports")
}
