package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	case err != nil:
		return nil, err
	}
	return CommitImport200JSONResponse(toPreview(p)), nil
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

func toPreview(p app.ImportPreview) ImportPreview {
	out := ImportPreview{
		Id: p.ID, Status: ImportPreviewStatus(p.Status), Source: ImportPreviewSource(p.Source),
		TotalRows: p.TotalRows, NewCount: p.NewCount, PossibleMatchCount: p.PossibleMatchCount,
		InvalidRows: make([]InvalidRow, 0, len(p.Problems)), IgnoredColumns: p.IgnoredColumns,
		UnknownFields: make([]string, 0, len(p.UnknownFields)),
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
		Id: string(p.ID), Species: cur.String(domain.FieldSpecies), Form: cur.String(domain.FieldForm),
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
