package user

import (
	"context"
)

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
