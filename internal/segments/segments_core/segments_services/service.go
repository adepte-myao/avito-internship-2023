package segments_services

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/segments_core/segments_domain"
	"avito-internship-2023/internal/segments/segments_core/segments_ports"
)

var (
	ErrSegmentNotExist   = errors.New("segment does not exist")
	ErrSlugAlreadyInUse  = errors.New("the slug is already in use")
	ErrUserDoesNotExist  = errors.New("user does not exist")
	ErrTooMuchParameters = errors.New("too much parameters")
)

type Service struct {
	logger              common.Logger
	providerContext     context.Context
	userServiceProvider segments_ports.UserServiceProvider
	userLocalProvider   segments_ports.UserProvider
	segmentsProvider    segments_ports.SegmentsProvider
	historyProvider     segments_ports.UserSegmentHistoryProvider
	deadlineAdder       segments_ports.DeadlineWorker
	fileStorage         segments_ports.FileStorage
}

func NewService(logger common.Logger, providerContext context.Context, userServiceProvider segments_ports.UserServiceProvider,
	userLocalProvider segments_ports.UserProvider, segmentsProvider segments_ports.SegmentsProvider, historyProvider segments_ports.UserSegmentHistoryProvider,
	deadlineAdder segments_ports.DeadlineWorker, fileStorage segments_ports.FileStorage) *Service {
	return &Service{
		logger:              logger,
		providerContext:     providerContext,
		userServiceProvider: userServiceProvider,
		userLocalProvider:   userLocalProvider,
		segmentsProvider:    segmentsProvider,
		historyProvider:     historyProvider,
		deadlineAdder:       deadlineAdder,
		fileStorage:         fileStorage,
	}
}

func (service *Service) ChangeSegmentsForUser(dto segments_ports.ChangeSegmentsForUserDTO) error {
	txCtx, err := service.segmentsProvider.BeginTransaction(service.providerContext)
	if err != nil {
		service.logger.Error(err)
		return err
	}
	defer func() {
		if err = service.userLocalProvider.Rollback(txCtx); err != nil {
			service.logger.Error(err)
		}
	}()

	err = errors.Join(
		service.validateUser(txCtx, dto.UserID),
		service.validateSegmentEntries(txCtx, dto.SegmentsToAdd, dto.SegmentsToRemove),
	)
	if err != nil {
		return err
	}

	userIDs := []string{dto.UserID}

	slugsToAdd := make([]string, len(dto.SegmentsToAdd))
	for i, segmentInfo := range dto.SegmentsToAdd {
		slugsToAdd[i] = segmentInfo.SegmentSlug
	}

	slugsToRemove := make([]string, len(dto.SegmentsToRemove))
	for i, segmentInfo := range dto.SegmentsToRemove {
		slugsToRemove[i] = segmentInfo.SegmentSlug
	}

	err = errors.Join(
		service.segmentsProvider.AddUsersToSegments(txCtx, userIDs, slugsToAdd),
		service.segmentsProvider.RemoveSegmentsForUser(txCtx, dto.UserID, slugsToRemove),
		service.addDeadlines(dto.UserID, dto.SegmentsToAdd),
	)
	if err != nil {
		return err
	}

	if err = service.segmentsProvider.Commit(txCtx); err != nil {
		service.logger.Error(err)
		return err
	}

	return nil
}

func (service *Service) validateUser(ctx context.Context, userID string) error {
	userExists, err := service.userLocalProvider.Exists(ctx, userID)
	if err != nil {
		return err
	}
	if !userExists {
		return ErrUserDoesNotExist
	}

	return nil
}

func (service *Service) validateSegmentEntries(ctx context.Context, toAdd []segments_ports.AddSegmentEntry, toRemove []segments_ports.RemoveSegmentEntry) error {
	allSlugs := make([]string, 0, len(toAdd)+len(toRemove))
	for _, segmentInfo := range toAdd {
		allSlugs = append(allSlugs, segmentInfo.SegmentSlug)
	}
	for _, segmentInfo := range toRemove {
		allSlugs = append(allSlugs, segmentInfo.SegmentSlug)
	}

	err := service.validateSegmentsExist(ctx, allSlugs)
	if err != nil {
		return err
	}

	for _, segmentInfo := range toAdd {
		if segmentInfo.SecondsToBeInSegment > 0 && !segmentInfo.DeadlineForStayingInSegment.IsZero() {
			return fmt.Errorf("%w: %s", ErrTooMuchParameters,
				"deadline must not be specified using both secondsToBeInSegment and deadlineForStayingInSegment")
		}
	}

	return nil
}

func (service *Service) validateSegmentsExist(ctx context.Context, slugs []string) error {
	segments, err := service.segmentsProvider.GetAllAsMap(ctx)
	if err != nil {
		service.logger.Error(err)
		return err
	}

	missedSlugs := make([]string, 0)
	for _, slug := range slugs {
		if _, ok := segments[slug]; !ok {
			missedSlugs = append(missedSlugs, slug)
		}
	}
	if len(missedSlugs) > 0 {
		err = fmt.Errorf("%w; missed segments: %s", ErrSegmentNotExist, strings.Join(missedSlugs, ", "))
		return err
	}

	return nil
}

func (service *Service) addDeadlines(userID string, segmentsToAdd []segments_ports.AddSegmentEntry) error {
	deadlines := make([]segments_domain.DeadlineEntry, 0)
	for _, segmentInfo := range segmentsToAdd {
		if segmentInfo.SecondsToBeInSegment == 0 && segmentInfo.DeadlineForStayingInSegment.IsZero() {
			continue
		}

		var deadlineTime time.Time
		if segmentInfo.SecondsToBeInSegment > 0 {
			deadlineTime = time.Now().Add(time.Second * time.Duration(segmentInfo.SecondsToBeInSegment))
		} else {
			deadlineTime = segmentInfo.DeadlineForStayingInSegment
		}

		deadlines = append(deadlines, segments_domain.DeadlineEntry{
			UserID:   userID,
			Slug:     segmentInfo.SegmentSlug,
			Deadline: deadlineTime,
		})
	}

	err := service.deadlineAdder.AddDeadlines(service.providerContext, deadlines)
	if err != nil {
		return err
	}

	return nil
}

func (service *Service) GetSegmentsForUser(dto segments_ports.GetSegmentsForUserDTO) (segments_ports.GetSegmentsForUserOutDTO, error) {
	slugs, err := service.segmentsProvider.GetForUser(service.providerContext, dto.UserID)
	if err != nil {
		return segments_ports.GetSegmentsForUserOutDTO{}, err
	}

	return segments_ports.GetSegmentsForUserOutDTO{Segments: slugs}, nil
}

func (service *Service) GetHistoryReportLink(dto segments_ports.GetSegmentsHistoryReportLinkDTO) (string, error) {
	entries, err := service.historyProvider.GetAllForUser(service.providerContext, dto.UserID, dto.Month, dto.Year)
	if err != nil {
		return "", err
	}

	toWrite := make([][]string, len(entries))
	for i, entry := range entries {
		toWrite[i] = []string{entry.Slug, entry.UserID, string(entry.ActionType), entry.LogTime.String()}
	}

	reportContent := &bytes.Buffer{}
	csvWriter := csv.NewWriter(reportContent)
	err = csvWriter.WriteAll(toWrite)
	if err != nil {
		return "", err
	}

	url, err := service.fileStorage.SaveCSVReportWithURLAccess(reportContent)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (service *Service) CreateSegment(dto segments_ports.CreateSegmentDTO) error {
	txCtx, err := service.segmentsProvider.BeginTransaction(service.providerContext)
	if err != nil {
		service.logger.Error(err)
		return err
	}
	defer func() {
		if err = service.userLocalProvider.Rollback(txCtx); err != nil {
			service.logger.Error(err)
		}
	}()

	err = service.createSegment(txCtx, dto)
	if err != nil {
		return err
	}

	if err = service.segmentsProvider.Commit(txCtx); err != nil {
		service.logger.Error(err)
		return err
	}

	if dto.PercentToFill > 0 {
		err = service.fillSegment(service.providerContext, dto)
	}
	if err != nil {
		return err
	}

	return nil
}

func (service *Service) createSegment(ctx context.Context, dto segments_ports.CreateSegmentDTO) error {
	err := service.validateSegmentsNotExist(ctx, []string{dto.Slug})
	if err != nil {
		return err
	}

	err = service.segmentsProvider.Create(ctx, segments_domain.Segment{Slug: dto.Slug})
	if err != nil {
		service.logger.Error(err)
		return err
	}

	return nil
}

func (service *Service) validateSegmentsNotExist(ctx context.Context, slugs []string) error {
	segments, err := service.segmentsProvider.GetAllAsMap(ctx)
	if err != nil {
		service.logger.Error(err)
		return err
	}

	usedSlugs := make([]string, 0)
	for _, slug := range slugs {
		if _, ok := segments[slug]; ok {
			usedSlugs = append(usedSlugs, slug)
		}
	}
	if len(usedSlugs) > 0 {
		err = fmt.Errorf("%w; used slugs: %s", ErrSlugAlreadyInUse, strings.Join(usedSlugs, ", "))
		return err
	}

	return nil
}

func (service *Service) fillSegment(ctx context.Context, segmentInfo segments_ports.CreateSegmentDTO) error {
	userIDsToAdd, err := service.getPercentOfUsers(ctx, segmentInfo.PercentToFill)

	slugsToAdd := []string{segmentInfo.Slug}
	err = service.segmentsProvider.AddUsersToSegments(ctx, userIDsToAdd, slugsToAdd)
	if err != nil {
		return err
	}

	return nil
}

func (service *Service) getPercentOfUsers(ctx context.Context, percent float64) ([]string, error) {
	users, err := service.userLocalProvider.GetAll(ctx)
	if err != nil {
		service.logger.Error(err)
		return nil, err
	}

	userIDs := make([]string, 0)
	partition := percent / 100
	for _, user := range users {
		if user.Status != segments_domain.Active {
			continue
		}

		if rand.Float64() < partition {
			userIDs = append(userIDs, user.Id)
		}
	}

	return userIDs, nil
}

func (service *Service) RemoveSegment(dto segments_ports.RemoveSegmentDTO) error {
	return service.segmentsProvider.Remove(service.providerContext, dto.Slug)
}

func (service *Service) CreateUser(dto segments_ports.CreateUserDTO) error {
	return service.userLocalProvider.Create(service.providerContext, segments_domain.User{Id: dto.UserID, Status: segments_domain.Active})
}

func (service *Service) UpdateUser(dto segments_ports.UpdateUserDTO) error {
	if err := segments_domain.ValidateUserStatus(dto.Status); err != nil {
		return err
	}

	return service.userLocalProvider.Update(service.providerContext, segments_domain.User{Id: dto.UserID, Status: dto.Status})
}

func (service *Service) RemoveUser(dto segments_ports.RemoveUserDTO) error {
	return service.userLocalProvider.Remove(service.providerContext, dto.UserID)
}

func (service *Service) ProcessUserAction(userID string) {
	userStatus, err := service.userServiceProvider.GetStatus(userID)
	if errors.Is(err, segments_domain.ErrUserNotFound) {
		err = service.userLocalProvider.Remove(context.TODO(), userID)
		if err != nil {
			service.logger.Error(err)
		}

		return
	}
	if err != nil {
		service.logger.Error(err)
		return
	}

	transactionCtx, err := service.userLocalProvider.BeginTransaction(service.providerContext)
	if err != nil {
		service.logger.Error(err)
		return
	}
	defer func() {
		if err = service.userLocalProvider.Rollback(transactionCtx); err != nil {
			service.logger.Error(err)
		}
	}()

	exists, err := service.userLocalProvider.Exists(transactionCtx, userID)
	if err != nil {
		service.logger.Error(err)
		return
	}

	targetUser := segments_domain.User{Id: userID, Status: userStatus}
	if exists {
		err = service.userLocalProvider.Update(transactionCtx, targetUser)
	} else {
		err = service.userLocalProvider.Create(transactionCtx, targetUser)
	}
	if err != nil {
		service.logger.Error(err)
		return
	}

	if err = service.userLocalProvider.Commit(transactionCtx); err != nil {
		service.logger.Error(err)
		return
	}
}
