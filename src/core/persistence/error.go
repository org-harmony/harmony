package persistence

import "errors"

var (
	// ErrInsert is returned when an insert fails. It is used in the repository implementations and wraps the underlying (database) error.
	ErrInsert = errors.New("failed to insert")
	// ErrUpdate is returned when an update fails. It is used in the repository implementations and wraps the underlying (database) error.
	ErrUpdate = errors.New("failed to update")
	// ErrDelete is returned when a delete fails. It is used in the repository implementations and wraps the underlying (database) error.
	ErrDelete = errors.New("failed to delete")
	// ErrReadRow is returned when a row could not be read. It is used in the repository implementations and wraps the underlying (database) error.
	ErrReadRow = errors.New("failed to read row")
	// ErrNotFound is returned when a row could not be found. It is used in the repository implementations and wraps the underlying (database) error.
	ErrNotFound = errors.New("not found")
)
