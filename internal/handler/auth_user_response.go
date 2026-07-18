package handler

import "uptime_ng/internal/model"

type UserProfileResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type UserListResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

func userProfileResponse(user model.User) UserProfileResponse {
	return UserProfileResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}

func userListResponses(users []model.User) []UserListResponse {
	results := make([]UserListResponse, len(users))
	for i, user := range users {
		results[i] = UserListResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			Active:   user.Active,
		}
	}
	return results
}
