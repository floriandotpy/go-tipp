package models

import (
	"errors"
)

var (
	ErrNoRecord           = errors.New("models: no matching record found")
	ErrInvalidCredentials = errors.New("models: invalid credentials")
	ErrDuplicateEmail     = errors.New("models: duplicate email")
	ErrDuplicateName      = errors.New("models: duplicate user name")
	ErrInvalidInvite      = errors.New("models: invalid invite code")
	ErrNotLoggedIn        = errors.New("models: not logged in")
)
