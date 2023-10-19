package persistence

import "errors"

var (
	InsertError   = errors.New("failed to insert")
	DeleteError   = errors.New("failed to delete")
	ReadRowError  = errors.New("failed to read row")
	NotFoundError = errors.New("not found")
)
