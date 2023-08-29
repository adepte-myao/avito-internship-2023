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

type remover interface {
	RemoveUser(dto segments.RemoveUserDTO) error
}

type RemoveHandler struct {
	remover  remover
	validate *validator.Validate
}

func NewRemoveHandler(remover remover, validate *validator.Validate) *RemoveHandler {
	return &RemoveHandler{
		remover:  remover,
		validate: validate,
	}
}

// Handle of user_handlers/RemoveHandler
// @Tags user
// @Description Removes user from local storage and excludes him from all segments he has
// @Accept json
// @Param input body segments.RemoveUserDTO true "userID to remove"
// @Success 204
// @Router /segments/remove-user [delete]
func (handler *RemoveHandler) Handle(c *gin.Context) {
	var dto segments.RemoveUserDTO
	if err := json.NewDecoder(c.Request.Body).Decode(&dto); err != nil {
		err = fmt.Errorf("%w: %w", common.ErrJSONUnmarshalling, err)
		_ = c.Error(err)
		return
	}

	if err := handler.validate.Struct(dto); err != nil {
		_ = c.Error(err)
		return
	}

	if err := handler.remover.RemoveUser(dto); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
