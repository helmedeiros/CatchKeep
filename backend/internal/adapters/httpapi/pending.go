package httpapi

import (
	"context"
	"net/http"
)

// Operations declared in the contract whose use cases have not landed yet
// answer 501 with the standard error envelope.

func notImplemented() Error {
	return errorBody(codeNotImplemented, "This feature is not available yet.")
}

func (s *server) ListProfiles(context.Context, ListProfilesRequestObject) (ListProfilesResponseObject, error) {
	return ListProfilesdefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) CreateProfile(context.Context, CreateProfileRequestObject) (CreateProfileResponseObject, error) {
	return CreateProfiledefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) SignIn(context.Context, SignInRequestObject) (SignInResponseObject, error) {
	return SignIndefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) SignOut(context.Context, SignOutRequestObject) (SignOutResponseObject, error) {
	return SignOutdefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}

func (s *server) GetMe(context.Context, GetMeRequestObject) (GetMeResponseObject, error) {
	return GetMedefaultJSONResponse{StatusCode: http.StatusNotImplemented, Body: notImplemented()}, nil
}
