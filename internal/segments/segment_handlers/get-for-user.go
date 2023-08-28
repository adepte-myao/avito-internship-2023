package segment_handlers

import (
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type segmentsGetter interface {
	GetSegmentsForUser(dto segments.GetSegmentsForUserDTO) (segments.GetSegmentsForUserOutDTO, error)
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

func (handler *GetForUserHandler) Handle(c *gin.Context) {
	var dto segments.GetSegmentsForUserDTO
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
