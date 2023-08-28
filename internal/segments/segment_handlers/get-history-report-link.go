package segment_handlers

import (
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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

func (handler *GetHistoryReportLinkHandler) Handle(c *gin.Context) {
	var dto segments.GetSegmentsHistoryReportLinkDTO
	if err := c.ShouldBindQuery(&dto); err != nil {
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

	outDto := map[string]string{"reportLink": link}
	c.JSON(http.StatusOK, outDto)
}
