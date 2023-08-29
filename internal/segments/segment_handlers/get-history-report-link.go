package segment_handlers

import (
	"fmt"
	"net/http"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type getSegmentsHistoryReportLinkOutDTO struct {
	Link           string `json:"reportLink"`
	ExpirationTime string `json:"expirationTime"`
}

type reportLinkGetter interface {
	GetHistoryReportLink(dto segments.GetSegmentsHistoryReportLinkDTO) (string, error)
}

type GetHistoryReportLinkHandler struct {
	reportLinkGetter reportLinkGetter
	validate         *validator.Validate
}

func NewGetHistoryReportLinkHandler(reportLinkGetter reportLinkGetter, validate *validator.Validate) *GetHistoryReportLinkHandler {
	return &GetHistoryReportLinkHandler{
		reportLinkGetter: reportLinkGetter,
		validate:         validate,
	}
}

// Handle of segment_handlers/GetHistoryReportLinkHandler
// @Tags segment
// @Description Returns the link to the report that contains history of segment assignments for given user in given month, year. Link to the report expires in four hours.
// @Produce json
// @Param userID query string true "identifier of user which history to provide"
// @Param month query int true "number of month (1-12) for which history will be provided"
// @Param year query int true "year"
// @Success 200 {object} getSegmentsHistoryReportLinkOutDTO
// @Router /segments/get-history-report-link [get]
func (handler *GetHistoryReportLinkHandler) Handle(c *gin.Context) {
	var dto segments.GetSegmentsHistoryReportLinkDTO
	if err := c.BindQuery(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrBindFailed, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	link, err := handler.reportLinkGetter.GetHistoryReportLink(dto)
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
