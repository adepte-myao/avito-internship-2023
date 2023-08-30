package segment_handlers

import (
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/segments_core"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type segmentsGetter interface {
	GetSegmentsForUser(dto segments_core.GetSegmentsForUserDTO) (segments_core.GetSegmentsForUserOutDTO, error)
}

type GetForUserHandler struct {
	segmentsGetter segmentsGetter
	validate       *validator.Validate
}

func NewGetForUserHandler(segmentsGetter segmentsGetter, validate *validator.Validate) *GetForUserHandler {
	return &GetForUserHandler{
		segmentsGetter: segmentsGetter,
		validate:       validate,
	}
}

// Handle of segment_handlers/GetForUserHandler
// @Tags segment
// @Description Returns the list of segments for given user
// @Produce json
// @Param userID query string true "identifier of user which segments to provide"
// @Success 200 {object} segments.GetSegmentsForUserOutDTO
// @Router /segments/get-for-user [get]
func (handler *GetForUserHandler) Handle(c *gin.Context) {
	var dto segments_core.GetSegmentsForUserDTO
	if err := c.BindQuery(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrBindFailed, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	outDto, err := handler.segmentsGetter.GetSegmentsForUser(dto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, outDto)
}
