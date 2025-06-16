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

// ErrURLDeleted возвращается когда URL помечен как удаленный
type ErrURLDeleted struct{}

func (e *ErrURLDeleted) Error() string {
	return "URL is deleted"
}

// IsURLDeleted проверяет, является ли ошибка ошибкой удаленного URL
func IsURLDeleted(err error) bool {
	_, ok := err.(*ErrURLDeleted)
	return ok
}

// ErrDeleteChannelFull возвращается когда канал удаления переполнен
var ErrDeleteChannelFull = errors.New("delete channel is full, try again later")
