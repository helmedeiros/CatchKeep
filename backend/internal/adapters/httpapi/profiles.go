package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

const sessionCookie = "ck_session"

type tokenKey struct{}

// readSessionCookie puts the session token, if any, into the request context.
func readSessionCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			r = r.WithContext(context.WithValue(r.Context(), tokenKey{}, c.Value))
		}
		next.ServeHTTP(w, r)
	})
}

type deviceKey struct{}

// markDevice notes the address a request comes from, which sign-in limits
// wrong PINs by. Forwarded headers are ignored: anyone could set them.
func markDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), deviceKey{}, host)))
	})
}

func device(ctx context.Context) string {
	d, _ := ctx.Value(deviceKey{}).(string)
	return d
}

type adminKey struct{}

// markAdmin marks requests made on the host itself and addressed to it as
// localhost: only the computer running CatchKeep may create profiles and
// reset PINs (OD-07, decided). A forwarded request is never admin.
func markAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if err == nil && ip != nil && ip.IsLoopback() && slices.Contains(loopbackNames, hostName(r.Host)) && r.Header.Get("X-Forwarded-For") == "" {
			r = r.WithContext(context.WithValue(r.Context(), adminKey{}, true))
		}
		next.ServeHTTP(w, r)
	})
}

func isAdmin(ctx context.Context) bool {
	admin, _ := ctx.Value(adminKey{}).(bool)
	return admin
}

func adminOnly() Error {
	return errorBody(codeAdminOnly, "Only the computer running CatchKeep can do this.")
}

func (s *server) GetHost(ctx context.Context, _ GetHostRequestObject) (GetHostResponseObject, error) {
	out := GetHost200JSONResponse{Admin: isAdmin(ctx), Addresses: []string{}}
	if out.Admin && s.services.Addresses != nil {
		out.Addresses = append(out.Addresses, s.services.Addresses()...)
	}
	return out, nil
}

func (s *server) ResetPin(ctx context.Context, req ResetPinRequestObject) (ResetPinResponseObject, error) {
	if !isAdmin(ctx) {
		return ResetPin403JSONResponse{ErrorJSONResponse(adminOnly())}, nil
	}
	err := s.services.Sessions.ResetPIN(ctx, domain.TrainerID(req.ProfileId), req.Body.Pin)
	switch {
	case errors.Is(err, domain.ErrInvalidPIN):
		return ResetPin422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case errors.Is(err, domain.ErrTrainerNotFound):
		return ResetPin404JSONResponse(errorBody(codeNotFound, "No such profile.")), nil
	case err != nil:
		return nil, err
	}
	return ResetPin204Response{}, nil
}

func tokenFrom(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey{}).(string)
	return token
}

func (s *server) sessionCookie(value string, maxAge int) string {
	// Secure follows the serving scheme: the host serves plain HTTP on its
	// network (OD-04, decided), so it is off unless configured.
	c := &http.Cookie{ //nolint:gosec // Secure is set from config, see above
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.services.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
	return c.String()
}

func toProfile(p app.ProfileSummary) ProfileSummary {
	return ProfileSummary{Id: string(p.ID), DisplayName: p.DisplayName}
}

func (s *server) ListProfiles(ctx context.Context, _ ListProfilesRequestObject) (ListProfilesResponseObject, error) {
	list, err := s.services.Profiles.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(ListProfiles200JSONResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toProfile(p))
	}
	return out, nil
}

func (s *server) CreateProfile(ctx context.Context, req CreateProfileRequestObject) (CreateProfileResponseObject, error) {
	if !isAdmin(ctx) {
		return CreateProfile403JSONResponse{ErrorJSONResponse(adminOnly())}, nil
	}
	p, err := s.services.Profiles.Create(ctx, req.Body.DisplayName, req.Body.Pin)
	switch {
	case errors.Is(err, domain.ErrInvalidDisplayName), errors.Is(err, domain.ErrInvalidPIN):
		return CreateProfile422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case errors.Is(err, domain.ErrDisplayNameTaken):
		return CreateProfile409JSONResponse(errorBody(codeNameTaken, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return CreateProfile201JSONResponse(toProfile(p)), nil
}

func (s *server) SignIn(ctx context.Context, req SignInRequestObject) (SignInResponseObject, error) {
	if err := domain.ValidatePIN(req.Body.Pin); err != nil {
		return SignIn422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	}
	token, me, err := s.services.Sessions.SignIn(ctx, device(ctx), domain.TrainerID(req.Body.ProfileId), req.Body.Pin)
	var locked *domain.LockedOutError
	switch {
	case errors.As(err, &locked):
		slog.WarnContext(ctx, "sign-in refused: too many wrong PINs", "device", device(ctx), "profile", req.Body.ProfileId, "until", locked.Until)
		body := errorBody(codeRateLimited, "Too many wrong PINs. Try again later.")
		body.Error.Details = &map[string]any{"retryAfter": locked.Until.UTC().Format(time.RFC3339)}
		return SignIn429JSONResponse(body), nil
	case errors.Is(err, domain.ErrWrongPIN):
		return SignIn401JSONResponse{ErrorJSONResponse(errorBody(codePINInvalid, "Wrong PIN."))}, nil
	case err != nil:
		return nil, err
	}
	cookie := s.sessionCookie(token, int(domain.SessionIdleTimeout.Seconds()))
	return SignIn200JSONResponse{
		Body:    Me(toProfile(me)),
		Headers: SignIn200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *server) SignOut(ctx context.Context, _ SignOutRequestObject) (SignOutResponseObject, error) {
	token := tokenFrom(ctx)
	if token == "" {
		return SignOut401JSONResponse{ErrorJSONResponse(errorBody(codeUnauthenticated, "Not signed in."))}, nil
	}
	if err := s.services.Sessions.SignOut(ctx, token); err != nil {
		return nil, err
	}
	cookie := s.sessionCookie("", -1)
	return SignOut204Response{Headers: SignOut204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (s *server) GetMe(ctx context.Context, _ GetMeRequestObject) (GetMeResponseObject, error) {
	me, err := s.services.Sessions.Resolve(ctx, tokenFrom(ctx))
	if errors.Is(err, domain.ErrNoSession) {
		return GetMe401JSONResponse{ErrorJSONResponse(errorBody(codeUnauthenticated, "Not signed in."))}, nil
	}
	if err != nil {
		return nil, err
	}
	return GetMe200JSONResponse(toProfile(me)), nil
}

func (s *server) GetAutostart(ctx context.Context, _ GetAutostartRequestObject) (GetAutostartResponseObject, error) {
	if !isAdmin(ctx) {
		return GetAutostart403JSONResponse{ErrorJSONResponse(adminOnly())}, nil
	}
	state, err := s.services.Autostart.State()
	if err != nil {
		return nil, err
	}
	return GetAutostart200JSONResponse{Supported: state.Supported, Enabled: state.Enabled}, nil
}

func (s *server) SetAutostart(ctx context.Context, req SetAutostartRequestObject) (SetAutostartResponseObject, error) {
	if !isAdmin(ctx) {
		return SetAutostart403JSONResponse{ErrorJSONResponse(adminOnly())}, nil
	}
	state, err := s.services.Autostart.Set(req.Body.Enabled)
	switch {
	case errors.Is(err, app.ErrAutostartUnsupported):
		return SetAutostart409JSONResponse(errorBody(codeNoAutostart, "This computer can't start CatchKeep at login from here.")), nil
	case err != nil:
		return nil, err
	}
	return SetAutostart200JSONResponse{Supported: state.Supported, Enabled: state.Enabled}, nil
}
