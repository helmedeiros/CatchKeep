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
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/csvimport"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/httpapi"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/ids"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/launchd"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/netaddr"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/pinhash"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/refdata"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/web"
	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/buildinfo"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

const usage = `usage:
  catchkeep version                print the build version
  catchkeep serve [-addr host:port] [-db file] [-web-dir dir]
                                   serve the API and, if set, the built frontend`

const shutdownTimeout = 10 * time.Second

// Connection limits: a 10 MB export over slow Wi-Fi fits the read timeout,
// and slow or stuck connections are dropped instead of piling up.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 3 * time.Minute
	writeTimeout      = 3 * time.Minute
	idleTimeout       = 2 * time.Minute
	maxHeaderBytes    = 64 << 10
)

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
		dbPath := fs.String("db", envOr("CATCHKEEP_DB", defaultDBPath()), "SQLite database file (env CATCHKEEP_DB)")
		webDir := fs.String("web-dir", os.Getenv("CATCHKEEP_WEB_DIR"), "directory with the built frontend (env CATCHKEEP_WEB_DIR)")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		err := serve(ctx, serveConfig{addr: *addr, dbPath: *dbPath, webDir: *webDir}, func(a string) {
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
	addr   string
	dbPath string
	webDir string
}

// serve runs the HTTP API until ctx is canceled, then shuts down gracefully.
// ready is called with the bound address once the listener is open.
func serve(ctx context.Context, cfg serveConfig, ready func(addr string)) error {
	if err := os.MkdirAll(filepath.Dir(cfg.dbPath), 0o700); err != nil {
		return fmt.Errorf("create database folder: %w", err)
	}
	store, err := sqlite.Open(ctx, cfg.dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := newServer(ctx, newRootHandler(cfg, store, ln.Addr()))

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

// newServer serves h with the connection limits; requests inherit ctx.
func newServer(ctx context.Context, h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
}

// newRootHandler routes /api/ to the API and everything else to the frontend
// bundle when a web directory is configured.
func newRootHandler(cfg serveConfig, store *sqlite.Store, listen net.Addr) http.Handler {
	trainers := sqlite.NewTrainers(store)
	hasher := pinhash.NewArgon2()
	clock := ids.SystemClock{}
	ulids := ids.NewULID()
	collection := sqlite.NewCollection(store)
	teams := sqlite.NewTeams(store)
	api := httpapi.NewHandler(httpapi.Services{
		Health:     app.NewHealth(buildinfo.Version()),
		Profiles:   app.NewProfiles(trainers, hasher, ulids, clock),
		Sessions:   app.NewSessions(trainers, sqlite.NewSessions(store), hasher, ids.Tokens{}, clock),
		Imports:    app.NewImports(csvimport.NewPokeGenie(time.Local), csvimport.Generic{}, collection, collection, ulids, clock),
		Collection: app.NewCollection(collection, collection, clock),
		Keep:       app.NewKeep(sqlite.NewSetups(store), collection, refdata.NewSpecies(), teams),
		Teams:      app.NewTeams(teams, ulids),
		Addresses:  func() []string { return netaddr.URLs(listen, net.InterfaceAddrs) },
		HostNames:  func() []string { return netaddr.HostNames(listen, net.InterfaceAddrs, hostname()) },
		Autostart:  app.NewAutostart(loginItem(cfg)),
		Transfers:  app.NewTransfers(sqlite.NewTransfers(store), collection, collection, ulids, clock),
	})
	if cfg.webDir == "" {
		return api
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/", web.Handler(os.DirFS(cfg.webDir)))
	return mux
}

// loginItem starts CatchKeep at login on macOS with the settings it runs
// with now, as absolute paths since launchd starts it from "/". Elsewhere,
// or when a path cannot be resolved, it returns nil (unsupported).
func loginItem(cfg serveConfig) ports.Autostart {
	home, err := os.UserHomeDir()
	if runtime.GOOS != "darwin" || err != nil {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	db, err := filepath.Abs(cfg.dbPath)
	if err != nil {
		return nil
	}
	args := []string{exe, "serve", "-addr", cfg.addr, "-db", db}
	if cfg.webDir != "" {
		web, err := filepath.Abs(cfg.webDir)
		if err != nil {
			return nil
		}
		args = append(args, "-web-dir", web)
	}
	return launchd.New(home, args)
}

func hostname() string {
	name, _ := os.Hostname()
	return name
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultDBPath is <user config dir>/CatchKeep/catchkeep.db, falling back to
// the working directory if the config dir is unknown.
func defaultDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "catchkeep.db"
	}
	return filepath.Join(dir, "CatchKeep", "catchkeep.db")
}
