package segment_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type SegmentHandler struct {
	service  ports.SegmentsService
	validate *validator.Validate
}

func NewSegmentHandler(service ports.SegmentsService, validate *validator.Validate) *SegmentHandler {
	return &SegmentHandler{service: service, validate: validate}
}

// ChangeForUser of segment_handlers/SegmentHandler
// @Tags segment
// @Description Validates given segment changes and applies them
// @Accept json
// @Param input body ports.ChangeSegmentsForUserDTO true "userID, which segments to change, and segments lists to add/remove"
// @Success 204
// @Router /segments/change-for-user [post]
func (handler *SegmentHandler) ChangeForUser(c *gin.Context) {
	var dto ports.ChangeSegmentsForUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.ChangeSegmentsForUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Create of segment_handlers/SegmentHandler
// @Tags segment
// @Description Validates given segment creates it
// @Accept json
// @Param input body ports.CreateSegmentDTO true "slug and probability for each user to be added to segment"
// @Success 204
// @Router /segments/create [post]
func (handler *SegmentHandler) Create(c *gin.Context) {
	var dto ports.CreateSegmentDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.CreateSegment(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetForUser of segment_handlers/SegmentHandler
// @Tags segment
// @Description Returns the list of segments for given user
// @Produce json
// @Param userID query string true "identifier of user which segments to provide"
// @Success 200 {object} ports.GetSegmentsForUserOutDTO
// @Router /segments/get-for-user [get]
func (handler *SegmentHandler) GetForUser(c *gin.Context) {
	var dto ports.GetSegmentsForUserDTO
	if err := c.BindQuery(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrBindFailed, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	outDto, err := handler.service.GetSegmentsForUser(dto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, outDto)
}

// GetHistoryReportLink of segment_handlers/SegmentHandler
// @Tags segment
// @Description Returns the link to the report that contains history of segment assignments for given user in given month, year. Link to the report expires in four hours.
// @Produce json
// @Param userID query string true "identifier of user which history to provide"
// @Param month query int true "number of month (1-12) for which history will be provided"
// @Param year query int true "year"
// @Success 200 {object} getSegmentsHistoryReportLinkOutDTO
// @Router /segments/get-history-report-link [get]
func (handler *SegmentHandler) GetHistoryReportLink(c *gin.Context) {
	var dto ports.GetSegmentsHistoryReportLinkDTO
	if err := c.BindQuery(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrBindFailed, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	link, err := handler.service.GetHistoryReportLink(dto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	outDto := getSegmentsHistoryReportLinkOutDTO{
		Link:           link,
		ExpirationTime: time.Now().Add(4 * time.Hour).String(),
	}

	c.JSON(http.StatusOK, outDto)
}

// Remove of segment_handlers/SegmentHandler
// @Tags segment
// @Description Removes given segment and excludes all users from it
// @Accept json
// @Param input body ports.RemoveSegmentDTO true "slug of segment to remove"
// @Success 204
// @Router /segments/remove [delete]
func (handler *SegmentHandler) Remove(c *gin.Context) {
	var dto ports.RemoveSegmentDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.RemoveSegment(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
