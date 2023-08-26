package user_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type creator interface {
	CreateUser(dto segments.CreateUserDTO) error
}

type CreateHandler struct {
	creator  creator
	validate *validator.Validate
}

func NewCreateHandler(creator creator, validate *validator.Validate) *CreateHandler {
	return &CreateHandler{
		creator:  creator,
		validate: validate,
	}
}

func (handler *CreateHandler) Handle(c *gin.Context) {
	var dto segments.CreateUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.creator.CreateUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
