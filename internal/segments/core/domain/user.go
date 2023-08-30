package domain

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidUserStatus = errors.New("invalid user status")
)

type User struct {
	Id     string
	Status UserStatus
}

type UserStatus string

const (
	Active   UserStatus = "active"
	Excluded UserStatus = "excluded"
)

var possibleStatuses = []UserStatus{Active, Excluded}

func ValidateUserStatus(status UserStatus) error {
	for _, possibleStatus := range possibleStatuses {
		if status == possibleStatus {
			return nil
		}
	}

	return ErrInvalidUserStatus
}
