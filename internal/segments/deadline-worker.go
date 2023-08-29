package segments

import (
	"context"
	"time"

	"avito-internship-2023/internal/pkg/common"
)

type deadlineProvider interface {
	GetAllBefore(ctx context.Context, maxTime time.Time) ([]DeadlineEntry, error)
	Remove(ctx context.Context, toRemove []DeadlineEntry) error
}

type DeadlineWorker struct {
	logger           common.Logger
	providerCtx      context.Context
	deadlineProvider deadlineProvider
	segmentsProvider segmentsProvider
}

func NewDeadlineWorker(logger common.Logger, providerCtx context.Context, deadlineProvider deadlineProvider, segmentsProvider segmentsProvider) *DeadlineWorker {
	return &DeadlineWorker{logger: logger, providerCtx: providerCtx, deadlineProvider: deadlineProvider, segmentsProvider: segmentsProvider}
}

func (worker *DeadlineWorker) RemoveExceededUserSegments() error {
	deadlines, err := worker.deadlineProvider.GetAllBefore(worker.providerCtx, time.Now())
	if err != nil {
		worker.logger.Error(err)
		return err
	}

	deadlinesMap := deadlinesToUserIDMap(deadlines)
	for userID, slugsToRemove := range deadlinesMap {
		currentUserSegments, err := worker.segmentsProvider.GetForUser(worker.providerCtx, userID)
		if err != nil {
			worker.logger.Error(err)
			return err
		}

		slugsToRemove = excludeDeletedSegments(slugsToRemove, currentUserSegments)

		err = worker.segmentsProvider.RemoveSegmentsForUser(worker.providerCtx, userID, slugsToRemove)
		if err != nil {
			worker.logger.Error(err)
			return err
		}
	}

	err = worker.deadlineProvider.Remove(worker.providerCtx, deadlines)
	if err != nil {
		worker.logger.Error(err)
		return err
	}

	return nil
}

func deadlinesToUserIDMap(deadlines []DeadlineEntry) map[string][]string {
	outMap := make(map[string][]string)
	for _, deadlineEntry := range deadlines {
		slugs, ok := outMap[deadlineEntry.UserID]

		if ok {
			slugs = append(slugs, deadlineEntry.Slug)
		} else {
			slugs = []string{deadlineEntry.Slug}
		}

		outMap[deadlineEntry.UserID] = slugs
	}

	return outMap
}

func excludeDeletedSegments(initialSlice, existingSegments []string) []string {
	existingSegmentsMap := sliceToMap(existingSegments)

	filteredSlice := make([]string, 0)
	for _, slug := range initialSlice {
		if _, ok := existingSegmentsMap[slug]; ok {
			filteredSlice = append(filteredSlice, slug)
		}
	}

	return filteredSlice
}

func sliceToMap(slice []string) map[string]bool {
	out := make(map[string]bool, len(slice))

	for _, val := range slice {
		out[val] = true
	}

	return out
}
