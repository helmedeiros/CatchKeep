// Command catchkeep runs the CatchKeep backend.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/httpapi"
	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/buildinfo"
)

const usage = `usage:
  catchkeep version                print the build version
  catchkeep serve [-addr host:port] serve the API`

const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "version":
		_, _ = fmt.Fprintln(stdout, buildinfo.Version())
		return 0
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		fs.SetOutput(stderr)
		addr := fs.String("addr", "127.0.0.1:8080", "address to listen on")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		err := serve(ctx, serveConfig{addr: *addr}, func(a string) {
			_, _ = fmt.Fprintf(stdout, "listening on http://%s\n", a)
		})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "serve:", err)
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
}

type serveConfig struct {
	addr string
}

// serve runs the HTTP API until ctx is canceled, then shuts down gracefully.
// ready is called with the bound address once the listener is open.
func serve(ctx context.Context, cfg serveConfig, ready func(addr string)) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{
		Handler:           httpapi.NewHandler(httpapi.Services{Health: app.NewHealth(buildinfo.Version())}),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	ready(ln.Addr().String())

	select {
	case err := <-errs:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
