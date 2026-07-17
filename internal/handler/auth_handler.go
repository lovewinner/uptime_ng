package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/config"
	"uptime_ng/internal/model"
)

type AuthHandler struct {
	DB *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
}

type TokenResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	var user model.User
	if err := h.DB.Where("username = ? AND active = ?", req.Username, true).First(&user).Error; err != nil {
		errorResponse(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}

	if !model.CheckPasswordHash(req.Password, user.Password) {
		errorResponse(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}

	token, err := model.GenerateJWT(&user, config.AppConfig.JWT.Secret, config.AppConfig.JWT.ExpireHours)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate token")
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	var count int64
	h.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		// first user
	}

	var existing model.User
	if err := h.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		errorResponse(c, http.StatusConflict, "username_exists", "username already exists")
		return
	}

	hashedPassword, err := model.HashPassword(req.Password)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
		return
	}

	role := model.RoleUser
	if count == 0 {
		role = model.RoleAdmin
	}

	user := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Role:     role,
		Active:   true,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "user_create_failed", err.Error())
		return
	}

	token, err := model.GenerateJWT(&user, config.AppConfig.JWT.Secret, config.AppConfig.JWT.ExpireHours)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate token")
		return
	}

	c.JSON(http.StatusCreated, TokenResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (h *AuthHandler) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user model.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

type UserListResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	var users []model.User
	h.DB.Find(&users)

	results := make([]UserListResponse, len(users))
	for i, u := range users {
		results[i] = UserListResponse{
			ID:       u.ID,
			Username: u.Username,
			Role:     u.Role,
			Active:   u.Active,
		}
	}

	c.JSON(http.StatusOK, results)
}

type UpdateUserRequest struct {
	Role     *string `json:"role"`
	Active   *bool   `json:"active"`
	Password *string `json:"password"`
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := c.GetUint("user_id")

	var target model.User
	if err := h.DB.First(&target, userID).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Role != nil {
		if *req.Role != model.RoleAdmin && *req.Role != model.RoleUser {
			badRequest(c, "invalid_role", "role must be admin or user")
			return
		}
		if target.Role == model.RoleAdmin && *req.Role != model.RoleAdmin && h.isLastActiveAdmin(target.ID) {
			badRequest(c, "last_admin", "cannot remove the last active admin")
			return
		}
		updates["role"] = *req.Role
	}
	if req.Active != nil {
		if target.ID == currentUserID && !*req.Active {
			badRequest(c, "self_deactivate", "cannot deactivate yourself")
			return
		}
		if target.Role == model.RoleAdmin && !*req.Active && h.isLastActiveAdmin(target.ID) {
			badRequest(c, "last_admin", "cannot deactivate the last active admin")
			return
		}
		updates["active"] = *req.Active
	}
	if req.Password != nil {
		if len(*req.Password) < 6 {
			badRequest(c, "invalid_password", "password must be at least 6 characters")
			return
		}
		hashedPassword, err := model.HashPassword(*req.Password)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
			return
		}
		updates["password"] = hashedPassword
	}
	if len(updates) == 0 {
		badRequest(c, "empty_update", "no updates provided")
		return
	}

	if err := h.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "user_update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (h *AuthHandler) isLastActiveAdmin(userID uint) bool {
	var count int64
	h.DB.Model(&model.User{}).
		Where("role = ? AND active = ?", model.RoleAdmin, true).
		Where("id <> ?", userID).
		Count(&count)
	return count == 0
}
