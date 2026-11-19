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
		Total: sum.Total, Kept: sum.Kept, Surplus: sum.Surplus, NeedsConfirmation: sum.NeedsConfirmation, Protected: sum.Protected,
		Minimum: sum.Minimum, RareMinimum: sum.RareMinimum, Groups: make([]VariantGroup, 0, len(sum.Groups)),
	}
	for _, g := range sum.Groups {
		vg := VariantGroup{
			Species: g.Species, Form: g.Form, Shiny: g.Shiny, Costume: g.Costume, Background: g.Background,
			Dynamax: g.Dynamax, Gigantamax: g.Gigantamax, Lucky: g.Lucky, Shadow: g.Shadow, Purified: g.Purified,
			Rare: g.Rare, Default: g.Default, TradeBonus: g.TradeBonus, Rule: VariantGroupRule(g.Rule), Count: g.Count, Minimum: g.Minimum, Keep: g.Keep,
			KeptIds: make([]string, 0, len(g.KeptIDs)), SurplusIds: make([]string, 0, len(g.SurplusIDs)),
			ProtectedIds: make([]string, 0, len(g.ProtectedIDs)),
		}
		for _, id := range g.ProtectedIDs {
			vg.ProtectedIds = append(vg.ProtectedIds, string(id))
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

func (s *server) GetCapacityBands(ctx context.Context, _ GetCapacityBandsRequestObject) (GetCapacityBandsResponseObject, error) {
	if _, err := s.trainer(ctx); errors.Is(err, errNotSignedIn) {
		return GetCapacityBands401JSONResponse{unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}
	out := CapacityBands{Bands: []CapacityBand{}}
	for _, b := range s.services.Keep.CapacityBands() {
		out.Bands = append(out.Bands, CapacityBand{UpTo: b.UpTo, Ordinary: b.Ordinary, Rare: b.Rare})
	}
	return GetCapacityBands200JSONResponse(out), nil
}

func (s *server) GetSpeciesRules(ctx context.Context, _ GetSpeciesRulesRequestObject) (GetSpeciesRulesResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetSpeciesRules401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	setup, err := s.services.Keep.Setup(ctx, trainer)
	if errors.Is(err, domain.ErrSetupNotFound) {
		return GetSpeciesRules409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	}
	if err != nil {
		return nil, err
	}
	return GetSpeciesRules200JSONResponse(toSpeciesRules(setup.Species)), nil
}

func (s *server) SaveSpeciesRules(ctx context.Context, req SaveSpeciesRulesRequestObject) (SaveSpeciesRulesResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return SaveSpeciesRules401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	rules := make([]domain.SpeciesRule, 0, len(req.Body.Rules))
	for _, r := range req.Body.Rules {
		rules = append(rules, domain.SpeciesRule{Species: r.Species, Minimum: r.Minimum, Favorite: r.Favorite})
	}
	err = s.services.Keep.SaveSpeciesRules(ctx, trainer, rules)
	switch {
	case errors.Is(err, domain.ErrSetupNotFound):
		return SaveSpeciesRules409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	case errors.Is(err, domain.ErrInvalidSetup):
		return SaveSpeciesRules422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return SaveSpeciesRules200JSONResponse(toSpeciesRules(rules)), nil
}

func toSpeciesRules(rules []domain.SpeciesRule) SpeciesRules {
	out := SpeciesRules{Rules: make([]SpeciesRule, 0, len(rules))}
	for _, r := range rules {
		out.Rules = append(out.Rules, SpeciesRule{Species: r.Species, Minimum: r.Minimum, Favorite: r.Favorite})
	}
	return out
}
