// Package httpapi is the driving HTTP adapter. It implements the server
// interface generated from api/openapi.yaml and maps DTOs to use cases.
package httpapi

import (
	"context"
	"net/http"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
)

// BaseURL is the path prefix every API route is served under.
const BaseURL = "/api/v1"

// Services are the use cases the HTTP API exposes.
type Services struct {
	Health     *app.Health
	Profiles   *app.Profiles
	Sessions   *app.Sessions
	Imports    *app.Imports
	Collection *app.Collection

	// SecureCookies marks the session cookie Secure. Enable it whenever the
	// API is served over HTTPS.
	SecureCookies bool
}

// NewHandler returns the HTTP handler for the whole API.
func NewHandler(services Services) http.Handler {
	strict := NewStrictHandlerWithOptions(&server{services: services}, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeRequestError,
		ResponseErrorHandlerFunc: writeInternalError,
	})
	return HandlerWithOptions(strict, StdHTTPServerOptions{
		BaseURL:          BaseURL,
		BaseRouter:       http.NewServeMux(),
		Middlewares:      []MiddlewareFunc{readSessionCookie},
		ErrorHandlerFunc: writeRequestError,
	})
}

type server struct {
	services Services
}

var _ StrictServerInterface = (*server)(nil)

func (s *server) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	report := s.services.Health.Check(ctx)
	return GetHealth200JSONResponse{
		Status:  HealthStatus(report.Status),
		Version: report.Version,
	}, nil
}
