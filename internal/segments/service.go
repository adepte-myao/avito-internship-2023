package segments

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"avito-internship-2023/internal/pkg/common"
)

var (
	ErrSegmentNotExist   = errors.New("segment does not exist")
	ErrSlugAlreadyInUse  = errors.New("the slug is already in use")
	ErrUserDoesNotExist  = errors.New("user does not exist")
	ErrTooMuchParameters = errors.New("too much parameters")
)

type userServiceProvider interface {
	GetStatus(userID string) (UserStatus, error)
}

type transactionHelper interface {
	BeginTransaction(ctx context.Context) (context.Context, error)
	Rollback(ctx context.Context) error
	Commit(ctx context.Context) error
}

type userLocalProvider interface {
	transactionHelper
	Create(ctx context.Context, user User) error
	GetAll(ctx context.Context) ([]User, error)
	Exists(ctx context.Context, userID string) (bool, error)
	Remove(ctx context.Context, userID string) error
	Update(ctx context.Context, user User) error
}

type segmentsProvider interface {
	transactionHelper
	GetAllAsMap(ctx context.Context) (map[string]Segment, error)
	GetForUser(ctx context.Context, userID string) ([]string, error)
	Create(ctx context.Context, segment Segment) error
	Remove(ctx context.Context, slug string) error
	AddUsersToSegments(ctx context.Context, userIDs, slugs []string) error
	RemoveSegmentsForUser(ctx context.Context, userID string, slugsToRemove []string) error
}

type userSegmentHistoryProvider interface {
	GetAllForUser(ctx context.Context, userID string, month, year int) ([]UserSegmentHistoryEntry, error)
}

type deadlineAdder interface {
	AddDeadlines(ctx context.Context, deadlines []DeadlineEntry) error
}

type Service struct {
	logger              common.Logger
	providerContext     context.Context
	userServiceProvider userServiceProvider
	userLocalProvider   userLocalProvider
	segmentsProvider    segmentsProvider
	historyProvider     userSegmentHistoryProvider
	deadlineAdder       deadlineAdder
}

func NewService(logger common.Logger, providerContext context.Context, userServiceProvider userServiceProvider,
	userLocalProvider userLocalProvider, segmentsProvider segmentsProvider, historyProvider userSegmentHistoryProvider,
	deadlineAdder deadlineAdder) *Service {
	return &Service{
		logger:              logger,
		providerContext:     providerContext,
		userServiceProvider: userServiceProvider,
		userLocalProvider:   userLocalProvider,
		segmentsProvider:    segmentsProvider,
		historyProvider:     historyProvider,
		deadlineAdder:       deadlineAdder,
	}
}

func (service *Service) ChangeSegmentsForUser(dto ChangeSegmentsForUserDTO) error {
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

func (service *Service) validateSegmentEntries(ctx context.Context, toAdd []AddSegmentEntry, toRemove []RemoveSegmentEntry) error {
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

func (service *Service) addDeadlines(userID string, segmentsToAdd []AddSegmentEntry) error {
	deadlines := make([]DeadlineEntry, 0)
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

		deadlines = append(deadlines, DeadlineEntry{
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

func (service *Service) GetSegmentsForUser(dto GetSegmentsForUserDTO) (GetSegmentsForUserOutDTO, error) {
	slugs, err := service.segmentsProvider.GetForUser(service.providerContext, dto.UserID)
	if err != nil {
		return GetSegmentsForUserOutDTO{}, err
	}

	return GetSegmentsForUserOutDTO{Segments: slugs}, nil
}

func (service *Service) GetHistoryReportLink(dto GetSegmentsHistoryReportLinkDTO) (string, error) {
	entries, err := service.historyProvider.GetAllForUser(service.providerContext, dto.UserID, dto.Month, dto.Year)
	if err != nil {
		return "", err
	}

	// TODO: dropbox integration?
	return fmt.Sprint(entries), nil
}

func (service *Service) CreateSegment(dto CreateSegmentDTO) error {
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

func (service *Service) createSegment(ctx context.Context, dto CreateSegmentDTO) error {
	err := service.validateSegmentsNotExist(ctx, []string{dto.Slug})
	if err != nil {
		return err
	}

	err = service.segmentsProvider.Create(ctx, Segment{Slug: dto.Slug})
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

func (service *Service) fillSegment(ctx context.Context, segmentInfo CreateSegmentDTO) error {
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
		if user.Status != Active {
			continue
		}

		if rand.Float64() < partition {
			userIDs = append(userIDs, user.Id)
		}
	}

	return userIDs, nil
}

func (service *Service) RemoveSegment(dto RemoveSegmentDTO) error {
	return service.segmentsProvider.Remove(service.providerContext, dto.Slug)
}

func (service *Service) CreateUser(dto CreateUserDTO) error {
	return service.userLocalProvider.Create(service.providerContext, User{Id: dto.UserID, Status: Active})
}

func (service *Service) UpdateUser(dto UpdateUserDTO) error {
	if err := validateUserStatus(dto.Status); err != nil {
		return err
	}

	return service.userLocalProvider.Update(service.providerContext, User{Id: dto.UserID, Status: dto.Status})
}

func (service *Service) RemoveUser(dto RemoveUserDTO) error {
	return service.userLocalProvider.Remove(service.providerContext, dto.UserID)
}

func (service *Service) ProcessUserAction(dto UserActionDTO) {
	userStatus, err := service.userServiceProvider.GetStatus(dto.UserID)
	if errors.Is(err, ErrUserNotFound) {
		err = service.userLocalProvider.Remove(context.TODO(), dto.UserID)
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

	exists, err := service.userLocalProvider.Exists(transactionCtx, dto.UserID)
	if err != nil {
		service.logger.Error(err)
		return
	}

	targetUser := User{Id: dto.UserID, Status: userStatus}
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
