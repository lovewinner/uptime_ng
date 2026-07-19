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

	c.JSON(http.StatusOK, tokenResponseFromUser(user, token))
}

func (h *AuthHandler) Register(c *gin.Context) {
	if !IsRegistrationOpen(h.DB) {
		errorResponse(c, http.StatusForbidden, "registration_closed", "registration is currently disabled")
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	var count int64
	if err := h.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "user_count_failed", err.Error())
		return
	}

	var existing model.User
	if err := h.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		errorResponse(c, http.StatusConflict, "username_exists", "username already exists")
		return
	} else if !isRecordNotFound(err) {
		errorResponse(c, http.StatusInternalServerError, "user_lookup_failed", err.Error())
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

	c.JSON(http.StatusCreated, tokenResponseFromUser(user, token))
}

func (h *AuthHandler) Profile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user model.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		lookupErrorResponse(c, err, "user_not_found", "user not found", "user_lookup_failed")
		return
	}

	c.JSON(http.StatusOK, userProfileResponse(user))
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.DB.Find(&users).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "user_list_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, userListResponses(users))
}

type UpdateUserRequest struct {
	Role     *string `json:"role"`
	Active   *bool   `json:"active"`
	Password *string `json:"password"`
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	userID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_user_id", "invalid user id")
		return
	}
	currentUserID := c.GetUint("user_id")

	var target model.User
	if err := h.DB.First(&target, userID).Error; err != nil {
		lookupErrorResponse(c, err, "user_not_found", "user not found", "user_lookup_failed")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	lastActiveAdmin := false
	if target.Role == model.RoleAdmin {
		var err error
		lastActiveAdmin, err = h.isLastActiveAdmin(target.ID)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "admin_count_failed", err.Error())
			return
		}
	}
	plan, validationErr := planUserUpdate(req, target, currentUserID, lastActiveAdmin)
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	hashedPassword := ""
	if plan.password != nil {
		var err error
		hashedPassword, err = model.HashPassword(*plan.password)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
			return
		}
	}

	updates := plan.fields(hashedPassword)
	if err := h.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "user_update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (h *AuthHandler) isLastActiveAdmin(userID uint) (bool, error) {
	var count int64
	if err := h.DB.Model(&model.User{}).
		Where("role = ? AND active = ?", model.RoleAdmin, true).
		Where("id <> ?", userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}
