package segments_ports

import (
	"io"

	"avito-internship-2023/internal/segments/segments_core/segments_domain"
)

type UserServiceProvider interface {
	GetStatus(userID string) (segments_domain.UserStatus, error)
}

type FileStorage interface {
	SaveCSVReportWithURLAccess(context io.Reader) (string, error)
}
