// Package web serves the built frontend bundle. Client-side routes fall back
// to index.html; hashed files under assets/ are cached for a year.
package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const (
	indexFile   = "index.html"
	assetsDir   = "assets/"
	assetsCache = "public, max-age=31536000, immutable"
)

// contentPolicy lets the app load only its own files and the Pokémon
// pictures, and never be shown inside another site. Styles allow inline
// <style> because the toast library injects one.
const contentPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' https://raw.githubusercontent.com data:; connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// Handler serves site, falling back to index.html for paths that are not files.
func Handler(site fs.FS) http.Handler {
	files := http.FileServerFS(site)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentPolicy)
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == indexFile {
			serveIndex(w, r, site)
			return
		}
		if isFile(site, name) {
			if strings.HasPrefix(name, assetsDir) {
				w.Header().Set("Cache-Control", assetsCache)
			}
			files.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(name, assetsDir) || path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, site)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, site fs.FS) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, site, indexFile)
}

func isFile(site fs.FS, name string) bool {
	info, err := fs.Stat(site, name)
	return err == nil && !info.IsDir()
}
