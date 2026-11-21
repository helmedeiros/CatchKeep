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
	setup := domain.Setup{
		Capacity: req.Body.Capacity, Goals: make([]domain.Goal, 0, len(req.Body.Goals)),
		InGameCount: deref(req.Body.InGameCount), Limit: deref(req.Body.Limit),
	}
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
			Rare: g.Rare, Category: category(g.Category), Default: g.Default, TradeBonus: g.TradeBonus, Rule: VariantGroupRule(g.Rule), Count: g.Count, Minimum: g.Minimum, Keep: g.Keep,
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
	if s.InGameCount > 0 {
		out.InGameCount = &s.InGameCount
	}
	if s.Limit > 0 {
		out.Limit = &s.Limit
	}
	return out
}

func (s *server) GetPlan(ctx context.Context, req GetPlanRequestObject) (GetPlanResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetPlan401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	target := domain.Target{Kind: domain.TargetKind(req.Params.Kind), Value: deref(req.Params.Value)}
	revision, err := s.services.Transfers.Revision(ctx, trainer)
	if err != nil {
		return nil, err
	}
	plan, err := s.services.Keep.Plan(ctx, trainer, target)
	switch {
	case errors.Is(err, domain.ErrSetupNotFound):
		return GetPlan409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	case errors.Is(err, domain.ErrInvalidTarget):
		return GetPlan422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	out := Plan{
		Use: StorageUse{
			Capacity: plan.Use.Capacity, Occupied: plan.Use.Occupied, Free: plan.Use.Free,
			Estimated: plan.Use.Estimated, Limit: plan.Use.Limit,
		},
		Kind: req.Params.Kind, Value: req.Params.Value, Wanted: plan.Wanted, MaxFeasible: plan.MaxFeasible,
		Shortfall: plan.Shortfall, Favorites: plan.Favorites, FavoritesUnknown: plan.FavoritesUnknown, Revision: revision,
		Transfers: make([]PlanItem, 0, len(plan.Transfers)),
	}
	out.Searches = make([]SpeciesSearch, 0, len(plan.Searches))
	for _, sp := range plan.Searches {
		search := SpeciesSearch{
			Species: sp.Species, Query: sp.Query, PokemonIds: ids(sp.IDs), Collisions: ids(sp.Collisions),
			Items: make([]ItemSearch, 0, len(sp.Items)),
		}
		for _, it := range sp.Items {
			if it.Query != "" || len(it.Collisions) > 0 {
				search.Items = append(search.Items, ItemSearch{PokemonId: string(it.ID), Query: it.Query, Collisions: ids(it.Collisions)})
			}
		}
		out.Searches = append(out.Searches, search)
	}
	for _, it := range plan.Transfers {
		out.Transfers = append(out.Transfers, PlanItem{
			PokemonId: string(it.ID), KeptInGroup: it.KeptInGroup, Rule: PlanItemRule(it.Rule), Favorite: toTri(it.Favorite),
		})
	}
	return GetPlan200JSONResponse(out), nil
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

func category(c domain.Category) *SpeciesCategory {
	if c == "" {
		return nil
	}
	out := SpeciesCategory(c)
	return &out
}

func (s *server) GetCategoryRules(ctx context.Context, _ GetCategoryRulesRequestObject) (GetCategoryRulesResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetCategoryRules401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	setup, err := s.services.Keep.Setup(ctx, trainer)
	if errors.Is(err, domain.ErrSetupNotFound) {
		return GetCategoryRules409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	}
	if err != nil {
		return nil, err
	}
	return GetCategoryRules200JSONResponse(toCategoryRules(setup.Categories)), nil
}

func (s *server) SaveCategoryRules(ctx context.Context, req SaveCategoryRulesRequestObject) (SaveCategoryRulesResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return SaveCategoryRules401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	rules := make([]domain.CategoryRule, 0, len(req.Body.Rules))
	for _, r := range req.Body.Rules {
		rules = append(rules, domain.CategoryRule{Category: domain.Category(r.Category), Minimum: r.Minimum, Favorite: r.Favorite})
	}
	err = s.services.Keep.SaveCategoryRules(ctx, trainer, rules)
	switch {
	case errors.Is(err, domain.ErrSetupNotFound):
		return SaveCategoryRules409JSONResponse(errorBody(codeSetupRequired, "Set up your storage and goals first.")), nil
	case errors.Is(err, domain.ErrInvalidSetup):
		return SaveCategoryRules422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return SaveCategoryRules200JSONResponse(toCategoryRules(rules)), nil
}

func toCategoryRules(rules []domain.CategoryRule) CategoryRules {
	out := CategoryRules{Rules: make([]CategoryRule, 0, len(rules))}
	for _, r := range rules {
		out.Rules = append(out.Rules, CategoryRule{Category: SpeciesCategory(r.Category), Minimum: r.Minimum, Favorite: r.Favorite})
	}
	return out
}

func toSpeciesRules(rules []domain.SpeciesRule) SpeciesRules {
	out := SpeciesRules{Rules: make([]SpeciesRule, 0, len(rules))}
	for _, r := range rules {
		out.Rules = append(out.Rules, SpeciesRule{Species: r.Species, Minimum: r.Minimum, Favorite: r.Favorite})
	}
	return out
}

func ids(in []domain.PokemonID) []string {
	out := make([]string, len(in))
	for i, id := range in {
		out[i] = string(id)
	}
	return out
}
