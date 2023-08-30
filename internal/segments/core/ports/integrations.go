package ports

import (
	"io"

	"avito-internship-2023/internal/segments/core/domain"
)

type UserServiceProvider interface {
	GetStatus(userID string) (domain.UserStatus, error)
}

type FileStorage interface {
	SaveCSVReportWithURLAccess(context io.Reader) (string, error)
}
