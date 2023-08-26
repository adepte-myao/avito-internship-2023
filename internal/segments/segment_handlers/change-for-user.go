package segment_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type segmentsChanger interface {
	ChangeSegmentsForUser(dto segments.ChangeSegmentsForUserDTO) error
}

type ChangeForUserHandler struct {
	segmentsChanger segmentsChanger
	validate        *validator.Validate
}

func NewChangeForUserHandler(segmentsChanger segmentsChanger, validate *validator.Validate) *ChangeForUserHandler {
	return &ChangeForUserHandler{
		segmentsChanger: segmentsChanger,
		validate:        validate,
	}
}

func (handler *ChangeForUserHandler) Handle(c *gin.Context) {
	var dto segments.ChangeSegmentsForUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.segmentsChanger.ChangeSegmentsForUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
