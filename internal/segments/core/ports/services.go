package ports

import (
	"context"

	"avito-internship-2023/internal/segments/core/domain"
)

type DeadlineWorker interface {
	AddDeadlines(ctx context.Context, deadlines []domain.DeadlineEntry) error
}

type SegmentsService interface {
	ChangeSegmentsForUser(dto ChangeSegmentsForUserDTO) error
	GetSegmentsForUser(dto GetSegmentsForUserDTO) (GetSegmentsForUserOutDTO, error)
	GetHistoryReportLink(dto GetSegmentsHistoryReportLinkDTO) (string, error)
	CreateSegment(dto CreateSegmentDTO) error
	RemoveSegment(dto RemoveSegmentDTO) error
	CreateUser(dto CreateUserDTO) error
	UpdateUser(dto UpdateUserDTO) error
	RemoveUser(dto RemoveUserDTO) error
	ProcessUserAction(userID string)
}
