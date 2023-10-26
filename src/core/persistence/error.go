package persistence

import "errors"

var (
	ErrInsert   = errors.New("failed to insert")
	ErrDelete   = errors.New("failed to delete")
	ErrReadRow  = errors.New("failed to read row")
	ErrNotFound = errors.New("not found")
)
