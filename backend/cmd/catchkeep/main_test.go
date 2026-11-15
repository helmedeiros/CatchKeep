package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun_VersionPrintsBuildVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(t.Context(), []string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "dev" {
		t.Fatalf("stdout = %q, want %q", got, "dev")
	}
}

func TestRun_UnknownCommandIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(t.Context(), []string{"bogus"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Fatalf("stderr = %q, want usage text", stderr.String())
	}
}

func TestServe_AnswersHealthAndStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	addrs := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		done <- serve(ctx, serveConfig{addr: "127.0.0.1:0", dbPath: filepath.Join(t.TempDir(), "test.db")}, func(addr string) { addrs <- addr })
	}()

	var addr string
	select {
	case addr = <-addrs:
	case err := <-done:
		t.Fatalf("serve exited before listening: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not start listening")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/api/v1/health", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	var body struct{ Status string }
	decodeErr := json.NewDecoder(resp.Body).Decode(&body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || decodeErr != nil || body.Status != "ok" {
		t.Fatalf("health: status %d, body %+v, decode error %v", resp.StatusCode, body, decodeErr)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v after cancel, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop after context cancel")
	}
}

func TestServe_ServesFrontendFromWebDirAlongsideAPI(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>catchkeep</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	addrs := make(chan string, 1)
	go func() {
		_ = serve(ctx, serveConfig{addr: "127.0.0.1:0", dbPath: filepath.Join(t.TempDir(), "test.db"), webDir: dir}, func(addr string) { addrs <- addr })
	}()
	addr := <-addrs

	for path, want := range map[string]string{
		"/":              "<html>catchkeep</html>",
		"/some/page":     "<html>catchkeep</html>",
		"/api/v1/health": `"status":"ok"`,
	} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+path, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(b), want) {
			t.Fatalf("GET %s = %d %q, want 200 containing %q", path, resp.StatusCode, b, want)
		}
	}
}

func TestServe_FailsWhenTheDatabaseCannotBeOpened(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := serve(t.Context(), serveConfig{addr: "127.0.0.1:0", dbPath: filepath.Join(blocker, "test.db")}, func(string) {
		t.Error("serve started listening without a database")
	})

	if err == nil {
		t.Fatal("serve returned nil, want error")
	}
}

func TestServe_CreatesTheDatabaseFolderOnFirstRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "new", "nested", "test.db")
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- serve(ctx, serveConfig{addr: "127.0.0.1:0", dbPath: dbPath}, func(string) { close(started) })
	}()

	select {
	case <-started:
	case err := <-done:
		t.Fatalf("serve failed: %v", err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("database file not created: %v", err)
	}
}
