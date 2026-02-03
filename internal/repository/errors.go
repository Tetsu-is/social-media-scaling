package repository

import "errors"

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateUser  = errors.New("user name is already used")
	ErrDuplicateTweet = errors.New("duplicate tweet")
	ErrNotImplemented = errors.New("not implemented")
)
