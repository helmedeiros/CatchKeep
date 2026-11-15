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
	Health *app.Health
}

// NewHandler returns the HTTP handler for the whole API.
func NewHandler(services Services) http.Handler {
	strict := NewStrictHandler(&server{services: services}, nil)
	return HandlerWithOptions(strict, StdHTTPServerOptions{
		BaseURL:    BaseURL,
		BaseRouter: http.NewServeMux(),
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
