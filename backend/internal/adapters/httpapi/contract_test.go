package httpapi_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

const contractPath = "../../../../api/openapi.yaml"

var loadRouter = sync.OnceValues(func() (routers.Router, error) {
	doc, err := openapi3.NewLoader().LoadFromFile(contractPath)
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(openapi3.NewLoader().Context); err != nil {
		return nil, err
	}
	// The contract declares a relative server URL; match on the full path.
	doc.Servers = openapi3.Servers{{URL: "http://example.com/api/v1"}}
	return gorillamux.NewRouter(doc)
})

// assertMatchesContract fails the test if the recorded response does not
// conform to api/openapi.yaml for the given request.
func assertMatchesContract(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	router, err := loadRouter()
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	matchReq := req.Clone(req.Context())
	matchReq.URL.Scheme, matchReq.URL.Host, matchReq.Host = "http", "example.com", "example.com"
	route, params, err := router.FindRoute(matchReq)
	if err != nil {
		t.Fatalf("route %s %s not in contract: %v", req.Method, req.URL.Path, err)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: matchReq, PathParams: params, Route: route},
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}
	if err := openapi3filter.ValidateResponse(req.Context(), input); err != nil {
		t.Fatalf("response violates contract: %v", err)
	}
}
