package web

import "net/http"

// HandlerError is issued by a controller's handler to the client.
// It contains the error, status code, and message to be issued.
//
// It is safe to display the message to the client.
//
// If Internal is true, the error is an internal server error and should not be displayed to the client.
// It should be assumed that the error has been logged if it is internal.
type HandlerError struct {
	Err      error
	Internal bool
	Status   int
	Message  string
}

// ExtErr returns a new HandlerError with the provided error, status code, and message.
func ExtErr(err error, status int, message string) HandlerError {
	return HandlerError{
		Err:      err,
		Status:   status,
		Message:  message,
		Internal: false,
	}
}

// IntErr returns a new internal HandlerError with a valid status code.
func IntErr() HandlerError {
	return HandlerError{
		Internal: true,
		Status:   http.StatusInternalServerError,
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

	return e.Err.Error()
}
