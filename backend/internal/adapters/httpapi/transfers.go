package httpapi

import (
	"context"
	"errors"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func (s *server) ConfirmTransfers(ctx context.Context, req ConfirmTransfersRequestObject) (ConfirmTransfersResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return ConfirmTransfers401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	pokemon := make([]domain.PokemonID, len(req.Body.PokemonIds))
	for i, id := range req.Body.PokemonIds {
		pokemon[i] = domain.PokemonID(id)
	}
	revision := int64(app.AnyRevision)
	if req.Body.PlanRevision != nil {
		revision = *req.Body.PlanRevision
	}
	e, err := s.services.Transfers.Confirm(ctx, trainer, pokemon, req.Body.RequestId, revision)
	var gone *domain.NotActiveError
	switch {
	case errors.As(err, &gone):
		body := errorBody(codeAlreadyTransferred, "Some of these are no longer in your collection; nothing was marked.")
		body.Error.Details = &map[string]any{"pokemonIds": ids(gone.IDs)}
		return ConfirmTransfers409JSONResponse(body), nil
	case errors.Is(err, domain.ErrStaleRevision):
		return ConfirmTransfers409JSONResponse(errorBody(codeStaleRevision, "Your collection changed since this plan was made; look at the new plan first.")), nil
	case errors.Is(err, domain.ErrInvalidTransfer):
		return ConfirmTransfers422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return ConfirmTransfers201JSONResponse(toTransferEvent(e)), nil
}

func (s *server) UndoTransfer(ctx context.Context, req UndoTransferRequestObject) (UndoTransferResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return UndoTransfer401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	e, err := s.services.Transfers.Undo(ctx, trainer, req.EventId)
	switch {
	case errors.Is(err, domain.ErrTransferNotFound):
		return UndoTransfer404JSONResponse(errorBody(codeNotFound, "No such transfer.")), nil
	case errors.Is(err, domain.ErrAlreadyUndone):
		return UndoTransfer409JSONResponse(errorBody(codeAlreadyUndone, err.Error())), nil
	case errors.Is(err, domain.ErrInvalidTransfer):
		return UndoTransfer422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return UndoTransfer201JSONResponse(toTransferEvent(e)), nil
}

func (s *server) GetHistory(ctx context.Context, _ GetHistoryRequestObject) (GetHistoryResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetHistory401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	events, err := s.services.Transfers.History(ctx, trainer)
	if err != nil {
		return nil, err
	}
	out := History{Events: make([]TransferEvent, 0, len(events))}
	for _, e := range events {
		out.Events = append(out.Events, toTransferEvent(e))
	}
	return GetHistory200JSONResponse(out), nil
}

func toTransferEvent(e domain.TransferEvent) TransferEvent {
	out := TransferEvent{Id: e.ID, Kind: TransferEventKind(e.Kind), At: e.At, Pokemon: make([]TransferredPokemon, 0, len(e.Pokemon))}
	for _, p := range e.Pokemon {
		tp := TransferredPokemon{Id: string(p.ID), Species: p.Species}
		if p.HasCP {
			cp := p.CP
			tp.Cp = &cp
		}
		out.Pokemon = append(out.Pokemon, tp)
	}
	if e.Undoes != "" {
		out.Undoes = &e.Undoes
	}
	if e.UndoneBy != "" {
		out.UndoneBy = &e.UndoneBy
	}
	return out
}
