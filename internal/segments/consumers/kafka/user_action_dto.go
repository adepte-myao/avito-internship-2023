package kafka

type userActionDTO struct {
	UserID string `json:"userID" validate:"required"`
}
