package web

import (
	"context"
)

// HandlerError contains the error, status code, and message to be issued.
//
// It is safe to display the message to the client.
//
// If Internal is true, the error is an internal server error and should not be displayed to the client.
// It should be assumed that the error has been logged if it is internal.
type HandlerError struct {
	Internal bool
	Message  string
}

// ErrorTemplateData is the template data for the default error template.
type ErrorTemplateData struct {
	Ctx context.Context // Ctx is the request's/error's context used for translation.
	Err string          // Err is the error message.
}

// ExtErr constructs an error that is safe to display to the client.
func ExtErr(message string) *HandlerError {
	return &HandlerError{
		Message:  message,
		Internal: false,
	}
}

// IntErr constructs an internal server error.
func IntErr() *HandlerError {
	return &HandlerError{
		Internal: true,
	}
}

// Error returns the error message.
func (e *HandlerError) Error() string {
	if e.Internal {
		return "internal server error - please review the logs"
	}

	if e.Message != "" {
		return e.Message
	}

	return "unknown error"
}

// NewErrorTemplateData returns a new ErrorTemplateData.
func NewErrorTemplateData(ctx context.Context, err string) *ErrorTemplateData {
	return &ErrorTemplateData{
		Ctx: ctx,
		Err: err,
	}
}
