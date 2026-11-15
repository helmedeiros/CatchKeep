package domain

import "errors"

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
