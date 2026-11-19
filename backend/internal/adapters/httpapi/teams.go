package httpapi

import (
	"context"
	"errors"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

const msgMemberNotFound = "A team member is not in your collection."

func (s *server) ListTeams(ctx context.Context, _ ListTeamsRequestObject) (ListTeamsResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return ListTeams401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	teams, err := s.services.Teams.List(ctx, trainer)
	if err != nil {
		return nil, err
	}
	out := Teams{Teams: make([]Team, 0, len(teams))}
	for _, t := range teams {
		out.Teams = append(out.Teams, toTeam(t))
	}
	return ListTeams200JSONResponse(out), nil
}

func (s *server) CreateTeam(ctx context.Context, req CreateTeamRequestObject) (CreateTeamResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return CreateTeam401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	team, err := s.services.Teams.Create(ctx, trainer, req.Body.Name, toPokemonIDs(req.Body.MemberIds))
	switch {
	case errors.Is(err, domain.ErrTeamNameTaken):
		return CreateTeam409JSONResponse(errorBody(codeTeamNameTaken, err.Error())), nil
	case errors.Is(err, domain.ErrInvalidTeam):
		return CreateTeam422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case errors.Is(err, domain.ErrPokemonNotFound):
		return CreateTeam422JSONResponse(errorBody(codeValidationFailed, msgMemberNotFound)), nil
	case err != nil:
		return nil, err
	}
	return CreateTeam201JSONResponse(toTeam(team)), nil
}

func (s *server) UpdateTeam(ctx context.Context, req UpdateTeamRequestObject) (UpdateTeamResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return UpdateTeam401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	team, err := s.services.Teams.Update(ctx, trainer, domain.TeamID(req.TeamId), req.Body.Name, toPokemonIDs(req.Body.MemberIds))
	switch {
	case errors.Is(err, domain.ErrTeamNotFound):
		return UpdateTeam404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	case errors.Is(err, domain.ErrTeamNameTaken):
		return UpdateTeam409JSONResponse(errorBody(codeTeamNameTaken, err.Error())), nil
	case errors.Is(err, domain.ErrInvalidTeam):
		return UpdateTeam422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case errors.Is(err, domain.ErrPokemonNotFound):
		return UpdateTeam422JSONResponse(errorBody(codeValidationFailed, msgMemberNotFound)), nil
	case err != nil:
		return nil, err
	}
	return UpdateTeam200JSONResponse(toTeam(team)), nil
}

func (s *server) DeleteTeam(ctx context.Context, req DeleteTeamRequestObject) (DeleteTeamResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return DeleteTeam401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	err = s.services.Teams.Delete(ctx, trainer, domain.TeamID(req.TeamId))
	if errors.Is(err, domain.ErrTeamNotFound) {
		return DeleteTeam404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	}
	if err != nil {
		return nil, err
	}
	return DeleteTeam204Response{}, nil
}

func toTeam(t domain.Team) Team {
	out := Team{Id: string(t.ID), Name: t.Name, MemberIds: make([]string, 0, len(t.Members))}
	for _, m := range t.Members {
		out.MemberIds = append(out.MemberIds, string(m))
	}
	return out
}

func toPokemonIDs(ids []string) []domain.PokemonID {
	out := make([]domain.PokemonID, 0, len(ids))
	for _, id := range ids {
		out = append(out, domain.PokemonID(id))
	}
	return out
}
