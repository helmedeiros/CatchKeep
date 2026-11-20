package httpapi

import (
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

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
