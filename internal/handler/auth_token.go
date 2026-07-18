package handler

import "uptime_ng/internal/model"

func tokenResponseFromUser(user model.User, token string) TokenResponse {
	return TokenResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}
