package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// maxImportBytes bounds uploaded exports (spec SEC-09).
const maxImportBytes = 10 << 20

var errNotSignedIn = errors.New("not signed in")

// trainer resolves the signed-in trainer from the session cookie.
func (s *server) trainer(ctx context.Context) (domain.TrainerID, error) {
	me, err := s.services.Sessions.Resolve(ctx, tokenFrom(ctx))
	if errors.Is(err, domain.ErrNoSession) {
		return "", errNotSignedIn
	}
	if err != nil {
		return "", err
	}
	return me.ID, nil
}

func unauthenticated() ErrorJSONResponse {
	return ErrorJSONResponse(errorBody(codeUnauthenticated, "Not signed in."))
}

func (s *server) StageImport(ctx context.Context, req StageImportRequestObject) (StageImportResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return StageImport401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxImportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if len(body) > maxImportBytes {
		return StageImport413JSONResponse(errorBody(codePayloadTooLarge, "The file is larger than 10 MB.")), nil
	}
	p, err := s.services.Imports.Stage(ctx, trainer, bytes.NewReader(body))
	if errors.Is(err, domain.ErrUnrecognizedExport) {
		return StageImport422JSONResponse(errorBody(codeUnsupportedFile, err.Error())), nil
	}
	if err != nil {
		return nil, err
	}
	return StageImport201JSONResponse(toPreview(p)), nil
}

func (s *server) GetImport(ctx context.Context, req GetImportRequestObject) (GetImportResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return GetImport401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := s.services.Imports.Get(ctx, trainer, req.ImportId)
	if errors.Is(err, domain.ErrImportNotFound) {
		return GetImport404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	}
	if err != nil {
		return nil, err
	}
	return GetImport200JSONResponse(toPreview(p)), nil
}

func (s *server) CommitImport(ctx context.Context, req CommitImportRequestObject) (CommitImportResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return CommitImport401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := s.services.Imports.Commit(ctx, trainer, req.ImportId)
	switch {
	case errors.Is(err, domain.ErrImportNotFound):
		return CommitImport404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	case errors.Is(err, domain.ErrImportAlreadyCommitted):
		return CommitImport409JSONResponse(errorBody(codeConflict, err.Error())), nil
	case errors.Is(err, domain.ErrImportNeedsMapping):
		return CommitImport409JSONResponse(errorBody(codeNeedsMapping, "Map the file's columns first.")), nil
	case err != nil:
		return nil, err
	}
	return CommitImport200JSONResponse(toPreview(p)), nil
}

func (s *server) AnswerPossibleMatches(ctx context.Context, req AnswerPossibleMatchesRequestObject) (AnswerPossibleMatchesResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return AnswerPossibleMatches401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	answers := make([]domain.Resolution, 0, len(req.Body.Answers))
	for _, a := range req.Body.Answers {
		r, ok := toResolution(a)
		if !ok {
			return AnswerPossibleMatches422JSONResponse(errorBody(codeValidationFailed,
				fmt.Sprintf("Row %d: answer \"same\" or \"different\", and name the candidate when it is the same.", a.Row))), nil
		}
		answers = append(answers, r)
	}
	p, err := s.services.Imports.Resolve(ctx, trainer, req.ImportId, answers)
	switch {
	case errors.Is(err, domain.ErrImportNotFound):
		return AnswerPossibleMatches404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	case errors.Is(err, domain.ErrImportAlreadyCommitted):
		return AnswerPossibleMatches409JSONResponse(errorBody(codeConflict, err.Error())), nil
	case errors.Is(err, domain.ErrImportNeedsMapping):
		return AnswerPossibleMatches409JSONResponse(errorBody(codeNeedsMapping, "Map the file's columns first.")), nil
	case errors.Is(err, domain.ErrInvalidResolution):
		return AnswerPossibleMatches422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return AnswerPossibleMatches200JSONResponse(toPreview(p)), nil
}

func (s *server) MapImportColumns(ctx context.Context, req MapImportColumnsRequestObject) (MapImportColumnsResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return MapImportColumns401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	mapping := domain.ColumnMapping{}
	for _, e := range req.Body.Mapping {
		if _, dup := mapping[e.Column]; dup {
			return MapImportColumns422JSONResponse(errorBody(codeValidationFailed, fmt.Sprintf("Column %q is mapped twice.", e.Column))), nil
		}
		mapping[e.Column] = domain.Field(e.Field)
	}
	p, err := s.services.Imports.Map(ctx, trainer, req.ImportId, mapping)
	switch {
	case errors.Is(err, domain.ErrImportNotFound):
		return MapImportColumns404JSONResponse(errorBody(codeNotFound, err.Error())), nil
	case errors.Is(err, domain.ErrImportAlreadyCommitted):
		return MapImportColumns409JSONResponse(errorBody(codeConflict, err.Error())), nil
	case errors.Is(err, domain.ErrInvalidMapping):
		return MapImportColumns422JSONResponse(errorBody(codeValidationFailed, err.Error())), nil
	case err != nil:
		return nil, err
	}
	return MapImportColumns200JSONResponse(toPreview(p)), nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toResolution(a MatchAnswer) (domain.Resolution, bool) {
	switch {
	case a.Answer == MatchAnswerAnswerDifferent:
		return domain.Resolution{Line: a.Row}, true
	case a.Answer == MatchAnswerAnswerSame && a.SameAs != nil && *a.SameAs != "":
		return domain.Resolution{Line: a.Row, SameAs: *a.SameAs}, true
	default:
		return domain.Resolution{}, false
	}
}

func (s *server) ListCollection(ctx context.Context, _ ListCollectionRequestObject) (ListCollectionResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return ListCollection401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	list, err := s.services.Collection.List(ctx, trainer)
	if err != nil {
		return nil, err
	}
	out := Collection{Total: len(list), Pokemon: make([]Pokemon, 0, len(list))}
	for _, p := range list {
		out.Pokemon = append(out.Pokemon, toPokemon(p))
	}
	return ListCollection200JSONResponse(out), nil
}

func (s *server) ConfirmAbsent(ctx context.Context, req ConfirmAbsentRequestObject) (ConfirmAbsentResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return ConfirmAbsent401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	invalid := func(msg string) ConfirmAbsentResponseObject {
		return ConfirmAbsent422JSONResponse(errorBody(codeValidationFailed, msg))
	}
	species, form := strings.TrimSpace(deref(req.Body.Species)), strings.TrimSpace(deref(req.Body.Form))
	if (species == "") != (form == "") {
		return invalid("Name both the species and the form, or neither to confirm the whole collection."), nil
	}
	fields := make([]domain.Field, 0, len(req.Body.Features))
	for _, f := range req.Body.Features {
		if !f.Valid() {
			return invalid(domain.ErrNotConfirmable.Error()), nil
		}
		fields = append(fields, domain.Field(f))
	}
	n, err := s.services.Collection.ConfirmAbsent(ctx, trainer, species, form, fields)
	if errors.Is(err, domain.ErrNotConfirmable) {
		return invalid(err.Error()), nil
	}
	if err != nil {
		return nil, err
	}
	return ConfirmAbsent200JSONResponse{Updated: n}, nil
}

func toPreview(p app.ImportPreview) ImportPreview {
	out := ImportPreview{
		Id: p.ID, Status: ImportPreviewStatus(p.Status), Source: ImportPreviewSource(p.Source),
		TotalRows: p.TotalRows, NewCount: p.NewCount, UpdatedCount: p.UpdatedCount, PossibleMatchCount: p.PossibleMatchCount,
		PossibleMatches: make([]PossibleMatch, 0, len(p.Matches)),
		InvalidRows:     make([]InvalidRow, 0, len(p.Problems)), IgnoredColumns: p.IgnoredColumns,
		UnknownFields: make([]string, 0, len(p.UnknownFields)),
		NeedsMapping:  p.NeedsMapping,
		Columns:       make([]ImportColumn, 0, len(p.Columns)),
		Mapping:       make([]ColumnMapEntry, 0, len(p.Mapping)),
	}
	for _, c := range p.Columns {
		samples := c.Samples
		if samples == nil {
			samples = []string{}
		}
		out.Columns = append(out.Columns, ImportColumn{Name: c.Name, Samples: samples})
		if f, ok := p.Mapping[c.Name]; ok {
			out.Mapping = append(out.Mapping, ColumnMapEntry{Column: c.Name, Field: MappableField(f)})
		}
	}
	for _, m := range p.Matches {
		out.PossibleMatches = append(out.PossibleMatches, toPossibleMatch(m))
	}
	if out.IgnoredColumns == nil {
		out.IgnoredColumns = []string{}
	}
	for _, pr := range p.Problems {
		out.InvalidRows = append(out.InvalidRows, InvalidRow{Row: pr.Line, Message: pr.Message})
	}
	for _, f := range p.UnknownFields {
		out.UnknownFields = append(out.UnknownFields, string(f))
	}
	return out
}

func toPossibleMatch(m domain.PossibleMatch) PossibleMatch {
	incoming := domain.Pokemon{ID: domain.PokemonID(domain.RowRef(m.Row.Line)), Observations: m.Row.Observations}
	out := PossibleMatch{
		Row: m.Row.Line, Incoming: toPokemon(incoming), Exact: m.Exact, Answer: PossibleMatchAnswerPending,
		Candidates: make([]MatchCandidate, 0, len(m.Candidates)),
	}
	for _, c := range m.Candidates {
		mc := MatchCandidate{Origin: MatchCandidateOriginCollection, Pokemon: toPokemon(c.Pokemon)}
		mc.Pokemon.Id = c.Ref
		if c.Line != 0 {
			line := c.Line
			mc.Origin, mc.Row = MatchCandidateOriginImport, &line
		}
		out.Candidates = append(out.Candidates, mc)
	}
	switch r := m.Resolution; {
	case r == nil:
	case r.SameAs == "":
		out.Answer = PossibleMatchAnswerDifferent
	default:
		sameAs := r.SameAs
		out.Answer, out.SameAs = PossibleMatchAnswerSame, &sameAs
	}
	return out
}

func toPokemon(p domain.Pokemon) Pokemon {
	cur := p.Current()
	tri := func(f domain.Field) TriState {
		switch cur.Tri(f) {
		case domain.Yes:
			return Yes
		case domain.No:
			return No
		default:
			return Unknown
		}
	}
	intPtr := func(f domain.Field) *int {
		if v, ok := cur.Int(f); ok {
			return &v
		}
		return nil
	}
	out := Pokemon{
		Id: string(p.ID), Species: cur.String(domain.FieldSpecies), Form: cur.String(domain.FieldForm), Protected: p.Protected,
		Cp: intPtr(domain.FieldCP), Hp: intPtr(domain.FieldHP), DexNumber: intPtr(domain.FieldDexNumber),
		Attributes: Attributes{
			Shiny: tri(domain.FieldShiny), Lucky: tri(domain.FieldLucky), Shadow: tri(domain.FieldShadow),
			Purified: tri(domain.FieldPurified), Favorite: tri(domain.FieldFavorite), Costume: tri(domain.FieldCostume),
			Background: tri(domain.FieldBackground), Dynamax: tri(domain.FieldDynamax), Gigantamax: tri(domain.FieldGigantamax),
		},
	}
	if out.Form == "" {
		out.Form = "Normal"
	}
	switch cur.String(domain.FieldGender) {
	case "male":
		g := Male
		out.Gender = &g
	case "female":
		g := Female
		out.Gender = &g
	}
	if lvl, ok := cur.Float(domain.FieldLevel); ok {
		l := float32(lvl)
		out.Level = &l
	}
	if iv, ok := cur.IVs(); ok {
		out.Iv = &IVs{Attack: iv.Attack, Defense: iv.Defense, Stamina: iv.Stamina}
	}
	if d, err := time.Parse(time.DateOnly, cur.String(domain.FieldCatchDate)); err == nil {
		out.CatchDate = &openapi_types.Date{Time: d}
	}
	return out
}

func (s *server) SetProtection(ctx context.Context, req SetProtectionRequestObject) (SetProtectionResponseObject, error) {
	trainer, err := s.trainer(ctx)
	if errors.Is(err, errNotSignedIn) {
		return SetProtection401JSONResponse{unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}
	err = s.services.Collection.Protect(ctx, trainer, domain.PokemonID(req.PokemonId), req.Body.Protected)
	if errors.Is(err, domain.ErrPokemonNotFound) {
		return SetProtection404JSONResponse(errorBody(codeNotFound, "That Pokémon is not in your collection.")), nil
	}
	if err != nil {
		return nil, err
	}
	return SetProtection200JSONResponse(*req.Body), nil
}
