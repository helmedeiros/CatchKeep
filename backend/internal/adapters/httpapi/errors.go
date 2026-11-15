package httpapi

import (
	"crypto/rand"
	"encoding/hex"
)

// Error codes from spec §10.2.
const (
	codeNotImplemented = "NOT_IMPLEMENTED"
)

func errorBody(code, message string) Error {
	return Error{Error: ErrorDetail{Code: code, Message: message, CorrelationId: newCorrelationID()}}
}

func newCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
