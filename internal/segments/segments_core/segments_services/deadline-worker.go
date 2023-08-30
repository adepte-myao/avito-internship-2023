package segments_services

import (
	"context"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/segments_core/segments_domain"
	"avito-internship-2023/internal/segments/segments_core/segments_ports"
)

type DeadlineWorker struct {
	logger           common.Logger
	providerCtx      context.Context
	deadlineProvider segments_ports.DeadlineProvider
	segmentsProvider segments_ports.SegmentsProvider
}

func NewDeadlineWorker(logger common.Logger, providerCtx context.Context, deadlineProvider segments_ports.DeadlineProvider, segmentsProvider segments_ports.SegmentsProvider) *DeadlineWorker {
	return &DeadlineWorker{logger: logger, providerCtx: providerCtx, deadlineProvider: deadlineProvider, segmentsProvider: segmentsProvider}
}

// Start works as http.ListenAndServe: blocks calling routine and can return only non-nil error
func (worker *DeadlineWorker) Start(deadlineCheckPeriodInSeconds int) error {
	for {
		time.Sleep(time.Second * time.Duration(deadlineCheckPeriodInSeconds))

		err := worker.RemoveExceededUserSegments()
		if err != nil {
			return err
		}
	}
}

func (worker *DeadlineWorker) RemoveExceededUserSegments() error {
	execContext, cancelExec := context.WithTimeout(worker.providerCtx, 10*time.Second)
	defer cancelExec()

	deadlines, err := worker.deadlineProvider.GetAllBefore(execContext, time.Now())
	if err != nil {
		worker.logger.Error(err)
		return err
	}

	deadlinesMap := deadlinesToUserIDMap(deadlines)
	for userID, slugsToRemove := range deadlinesMap {
		currentUserSegments, err := worker.segmentsProvider.GetForUser(execContext, userID)
		if err != nil {
			worker.logger.Error(err)
			return err
		}

		slugsToRemove = excludeDeletedSegments(slugsToRemove, currentUserSegments)

		err = worker.segmentsProvider.RemoveSegmentsForUser(execContext, userID, slugsToRemove)
		if err != nil {
			worker.logger.Error(err)
			return err
		}
	}

	err = worker.deadlineProvider.Remove(execContext, deadlines)
	if err != nil {
		worker.logger.Error(err)
		return err
	}

	return nil
}

func deadlinesToUserIDMap(deadlines []segments_domain.DeadlineEntry) map[string][]string {
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
