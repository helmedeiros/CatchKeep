package domain

import (
	"errors"
	"time"
)

// ErrUnrecognizedExport means an uploaded file is not a supported export.
var ErrUnrecognizedExport = errors.New("this file is not a Poke Genie CSV export")

// ImportRow is one Pokémon read from an export, as observations.
type ImportRow struct {
	Line         int
	Observations []Observation
}

// ImportProblem explains a row that was skipped, or a value that was left
// unknown, while reading an export. Line is 1-based and counts the header.
type ImportProblem struct {
	Line    int
	Message string
	Skipped bool
}

// ParsedExport is everything read from one export file.
type ParsedExport struct {
	Source         string
	Rows           []ImportRow
	Problems       []ImportProblem
	IgnoredColumns []string
	// UnknownFields are features this export can never establish.
	UnknownFields []Field
}

// Errors returned by import use cases.
var (
	ErrImportNotFound         = errors.New("import not found")
	ErrImportAlreadyCommitted = errors.New("import was already committed")
)

// ImportStatus is the lifecycle state of an import.
type ImportStatus string

// Import states.
const (
	ImportStaged    ImportStatus = "staged"
	ImportCommitted ImportStatus = "committed"
)

// ImportBatch is one uploaded export, staged for review and later committed.
type ImportBatch struct {
	ID                 string
	TrainerID          TrainerID
	Status             ImportStatus
	Parsed             ParsedExport
	NewCount           int
	UpdatedCount       int
	PossibleMatchCount int
	// Resolutions are the trainer's duplicate review answers, by line.
	Resolutions map[int]Resolution
	CreatedAt   time.Time
	CommittedAt time.Time
}
