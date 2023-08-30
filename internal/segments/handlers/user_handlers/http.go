package user_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	service  ports.SegmentsService
	validate *validator.Validate
}

func NewUserHandler(service ports.SegmentsService, validate *validator.Validate) *UserHandler {
	return &UserHandler{service: service, validate: validate}
}

// Create of user_handlers/UserHandler
// @Tags user
// @Description Saves information about user in local storage
// @Accept json
// @Param input body ports.CreateUserDTO true "userID to save"
// @Success 204
// @Router /segments/create-user [post]
func (handler *UserHandler) Create(c *gin.Context) {
	var dto ports.CreateUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.CreateUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Remove of user_handlers/UserHandler
// @Tags user
// @Description Removes user from local storage and excludes him from all segments he has
// @Accept json
// @Param input body ports.RemoveUserDTO true "userID to remove"
// @Success 204
// @Router /segments/remove-user [delete]
func (handler *UserHandler) Remove(c *gin.Context) {
	var dto ports.RemoveUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.RemoveUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Update of user_handlers/UserHandler
// @Tags user
// @Description Updates user info
// @Accept json
// @Param input body ports.UpdateUserDTO true "userID and his status to update "
// @Success 204
// @Router /segments/update-user [put]
func (handler *UserHandler) Update(c *gin.Context) {
	var dto ports.UpdateUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.service.UpdateUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
