package persistence

// SessionStorage defines possible operations on a session storage while abstracting the underlying implementation.
type SessionStorage interface {
	Create(data string) (string, error)  // Create returns the ID of the newly created session.
	Read(id string) (string, error)      // Read returns the data associated with the session ID.
	Update(id string, data string) error // Update updates the data associated with the session ID.
	Delete(id string) error              // Delete deletes the session by ID.
}
