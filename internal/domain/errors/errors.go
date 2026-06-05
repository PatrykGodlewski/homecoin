package errors

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrAlreadyExists       = errors.New("already exists")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidInput        = errors.New("invalid input")
	ErrAlreadyInHousehold  = errors.New("user already belongs to a household")
	ErrNotInHousehold      = errors.New("user is not in a household")
	ErrInvalidSplit        = errors.New("invalid expense split")
	ErrBudgetExceeded      = errors.New("budget threshold exceeded")
)
