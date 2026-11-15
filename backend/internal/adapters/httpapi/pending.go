package httpapi

import (
	"context"
	"net/http"
)

// Operations declared in the contract whose use cases have not landed yet
// answer 501 with the standard error envelope.

const codeNotImplemented = "NOT_IMPLEMENTED"

func notImplemented() Error {
	return errorBody(codeNotImplemented, "This feature is not available yet.")
}

func (s *server) StageImport(context.Context, StageImportRequestObject) (StageImportResponseObject, error) {
	return StageImportdefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) GetImport(context.Context, GetImportRequestObject) (GetImportResponseObject, error) {
	return GetImportdefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) CommitImport(context.Context, CommitImportRequestObject) (CommitImportResponseObject, error) {
	return CommitImportdefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) ListCollection(context.Context, ListCollectionRequestObject) (ListCollectionResponseObject, error) {
	return ListCollectiondefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}
