package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Error codes from spec §10.2.
const (
	codeValidationFailed = "VALIDATION_FAILED"
	codeUnauthenticated  = "UNAUTHENTICATED"
	codePINInvalid       = "PIN_INVALID"
	codeRateLimited      = "RATE_LIMITED"
	codeNameTaken        = "DISPLAY_NAME_TAKEN"
	codeInternal         = "INTERNAL"
)

func errorBody(code, message string) Error {
	return Error{Error: ErrorDetail{Code: code, Message: message, CorrelationId: newCorrelationID()}}
}

func newCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeError(w http.ResponseWriter, status int, body Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeRequestError answers requests the generated layer could not decode.
func writeRequestError(w http.ResponseWriter, _ *http.Request, err error) {
	writeError(w, http.StatusUnprocessableEntity, errorBody(codeValidationFailed, "The request is not valid: "+err.Error()))
}

// writeInternalError answers unexpected failures without leaking details.
func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	body := errorBody(codeInternal, "Something went wrong. Check the host logs.")
	slog.ErrorContext(r.Context(), "request failed", "error", err, "correlationId", body.Error.CorrelationId,
		"method", r.Method, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, body)
}
