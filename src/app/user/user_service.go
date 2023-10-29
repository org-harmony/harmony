package user

import (
	"context"
)

func UpdateUser(ctx context.Context, user *User, toUpdate *ToUpdate, repo Repository) error {
	// validate to update user

	user.Firstname = toUpdate.Firstname
	user.Lastname = toUpdate.Lastname

	// update user in db

	// write update user into session

	// write update user into context

	// write update user into user

	return nil
}
