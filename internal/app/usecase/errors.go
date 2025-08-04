// Package usecase содержит бизнес-логику сервиса сокращения URL
package usecase

import "errors"

// ErrURLConflict представляет ошибку при попытке сохранить уже существующий URL
type ErrURLConflict struct {
	ExistingShortURL string
}

// Error реализует интерфейс error для ErrURLConflict
func (e *ErrURLConflict) Error() string {
	return "URL already exists"
}

// IsURLConflict проверяет, является ли ошибка конфликтом URL
func IsURLConflict(err error) (*ErrURLConflict, bool) {
	var conflictErr *ErrURLConflict
	if errors.As(err, &conflictErr) {
		return conflictErr, true
	}
	return nil, false
}

// ErrURLDeleted представляет ошибку при попытке доступа к удаленному URL
type ErrURLDeleted struct{}

// Error реализует интерфейс error для ErrURLDeleted
func (e *ErrURLDeleted) Error() string {
	return "URL is deleted"
}

// IsURLDeleted проверяет, является ли ошибка признаком удаленного URL
func IsURLDeleted(err error) bool {
	_, ok := err.(*ErrURLDeleted)
	return ok
}

// ErrDeleteChannelFull возвращается, когда канал удаления переполнен
var ErrDeleteChannelFull = errors.New("delete channel is full, try again later")
