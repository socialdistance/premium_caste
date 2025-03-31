package storage

import "errors"

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrAppNotFound  = errors.New("app not found")
	ErrAppList      = errors.New("no such apps")
	ErrorNoSuchKey  = errors.New("no such key")
)
