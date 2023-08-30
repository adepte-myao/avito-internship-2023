package segments_user_service

type userActionDTO struct {
	UserID string `json:"userID" validate:"required"`
}
