package user_kafka_consumers

type userActionDTO struct {
	UserID string `json:"userID" validate:"required"`
}
