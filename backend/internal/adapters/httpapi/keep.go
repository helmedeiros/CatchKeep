package httpapi

import (
	"context"
	"errors"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func (s *server) GetSetup(ctx context.Context, _ GetSetupRequestObject) (GetSetupResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetSetup401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	setup, err := s.services.Keep.Setup(ctx, trainer)
	if errors.Is(err, domain.ErrSetupNotFound) {
		return GetSetup404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	}
	if err != nil {
		return nil, err
	}
	return GetSetup200JSONResponse(toSetup(setup)), nil
}

func (s *server) SaveSetup(ctx context.Context, req SaveSetupRequestObject) (SaveSetupResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return SaveSetup401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	setup := domain.Setup{Capacity: req.Body.Capacity, Goals: make([]domain.Goal, 0, len(req.Body.Goals))}
	for _, g := range req.Body.Goals {
		setup.Goals = append(setup.Goals, domain.Goal(g))
	}
	err = s.services.Keep.SaveSetup(ctx, trainer, setup)
	if errors.Is(err, domain.ErrInvalidSetup) {
		return SaveSetup422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	}
	if err != nil {
		return nil, err
	}
	return SaveSetup200JSONResponse(toSetup(setup)), nil
}

func (s *server) GetKeepSummary(ctx context.Context, _ GetKeepSummaryRequestObject) (GetKeepSummaryResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetKeepSummary401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	sum, err := s.services.Keep.Summary(ctx, trainer)
	if errors.Is(err, domain.ErrSetupNotFound) {
		return GetKeepSummary409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	}
	if err != nil {
		return nil, err
	}
	out := KeepSummary{
		Total: sum.Total, Kept: sum.Kept, Surplus: sum.Surplus, NeedsConfirmation: sum.NeedsConfirmation,
		Minimum: sum.Minimum, RareMinimum: sum.RareMinimum, Groups: make([]VariantGroup, 0, len(sum.Groups)),
	}
	for _, g := range sum.Groups {
		vg := VariantGroup{
			Species: g.Species, Form: g.Form, Shiny: g.Shiny, Costume: g.Costume, Background: g.Background,
			Dynamax: g.Dynamax, Gigantamax: g.Gigantamax, Lucky: g.Lucky, Shadow: g.Shadow, Purified: g.Purified,
			Rare: g.Rare, Count: g.Count, Minimum: g.Minimum, Keep: g.Keep,
			KeptIds: make([]string, 0, len(g.KeptIDs)), SurplusIds: make([]string, 0, len(g.SurplusIDs)),
		}
		for _, id := range g.KeptIDs {
			vg.KeptIds = append(vg.KeptIds, string(id))
		}
		for _, id := range g.SurplusIDs {
			vg.SurplusIds = append(vg.SurplusIds, string(id))
		}
		out.Groups = append(out.Groups, vg)
	}
	return GetKeepSummary200JSONResponse(out), nil
}

func toSetup(s domain.Setup) Setup {
	out := Setup{Capacity: s.Capacity, Goals: make([]SetupGoals, 0, len(s.Goals))}
	for _, g := range s.Goals {
		out.Goals = append(out.Goals, SetupGoals(g))
	}
	return out
}
