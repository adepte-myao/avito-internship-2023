package segments_ports

import (
	"context"

	"avito-internship-2023/internal/segments/segments_core/segments_domain"
)

type DeadlineWorker interface {
	AddDeadlines(ctx context.Context, deadlines []segments_domain.DeadlineEntry) error
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
