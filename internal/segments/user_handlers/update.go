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

type updater interface {
	UpdateUser(dto segments.UpdateUserDTO) error
}

type UpdateHandler struct {
	updater  updater
	validate *validator.Validate
}

func NewUpdateHandler(updater updater, validate *validator.Validate) *UpdateHandler {
	return &UpdateHandler{
		updater:  updater,
		validate: validate,
	}
}

// Handle of user_handlers/UpdateHandler
// @Tags user
// @Description Updates user info
// @Accept json
// @Param input body segments.UpdateUserDTO true "userID and his status to update "
// @Success 204
// @Router /segments/update-user [put]
func (handler *UpdateHandler) Handle(c *gin.Context) {
	var dto segments.UpdateUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.updater.UpdateUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
