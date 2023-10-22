package persistence

import "errors"

var (
	// ErrInsert is returned when an insert operation fails.
	ErrInsert = errors.New("failed to insert")
	// ErrDelete is returned when a delete operation fails.
	ErrDelete = errors.New("failed to delete")
	// ErrReadRow is returned when a read operation of a single row fails.
	ErrReadRow = errors.New("failed to read row")
	// ErrNotFound is returned when a row was not found.
	ErrNotFound = errors.New("not found")
	// ErrSessionExpired is returned when a session has expired.
	ErrSessionExpired = errors.New("session expired")
)
