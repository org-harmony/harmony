package user

import (
	"context"
	"errors"
	"time"
)

// ErrHardSessionExpiry is returned when a session has expired and the user has not logged in for more than 24 hours.
// Hard session expiry happens when the softly expired session could not be (further) extended.
var ErrHardSessionExpiry = errors.New("session is expired and user has not logged in for more than 24 hours")

// UpdateUser updates the user in the database and the session.
// It is a service function agnostic from the calling controller.
func UpdateUser(ctx context.Context, toUpdate *ToUpdate, session *Session, repo Repository, sessionStore SessionRepository) (*User, error) {
	update, err := repo.Update(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	session.Payload = *update // update the user in the session payload
	err = sessionStore.Write(ctx, session.ID, session)
	if err != nil {
		return nil, err
	}

	return update, nil
}

// TryExtendSession tries to extend the passed in session to the passed in duration.
// If the session is hard expired it returns ErrHardSessionExpiry. Hard expired is determined through Session.IsHardExpired.
func TryExtendSession(ctx context.Context, session *Session, duration time.Duration, sessionStore SessionRepository) error {
	if session.IsHardExpired() {
		return ErrHardSessionExpiry
	}

	now := time.Now()
	session.ExpiresAt = now.Add(duration)
	session.Meta.ExtendedAt = &now

	err := sessionStore.Write(ctx, session.ID, session)
	if err != nil {
		return err
	}

	return nil
}
