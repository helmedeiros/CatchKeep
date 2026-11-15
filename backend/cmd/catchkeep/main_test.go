package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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
		done <- serve(ctx, serveConfig{addr: "127.0.0.1:0"}, func(addr string) { addrs <- addr })
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
