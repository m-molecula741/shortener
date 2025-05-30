package usecase

import "errors"

// ErrURLConflict возвращается когда URL уже существует в базе данных
type ErrURLConflict struct {
	ExistingShortURL string
}

func (e *ErrURLConflict) Error() string {
	return "URL already exists"
}

func IsURLConflict(err error) (*ErrURLConflict, bool) {
	var conflictErr *ErrURLConflict
	if errors.As(err, &conflictErr) {
		return conflictErr, true
	}
	return nil, false
}
