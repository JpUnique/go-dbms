package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
}

// constructor
func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// ==============================
// REGISTER
// ==============================
func (h *AuthHandler) Register(c *gin.Context) {

	var req struct {
		Email      string  `json:"email" binding:"required,email"`
		Password   string  `json:"password" binding:"required"`
		Name       string  `json:"name" binding:"required"`
		Department *string `json:"department"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Register(
		c.Request.Context(),
		req.Email,
		req.Password,
		req.Name,
	)
	if err != nil {

		switch err {
		case utils.ErrAlreadyExists:
			utils.Error(c, http.StatusConflict, "user already exists")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	utils.Created(c, gin.H{
		"user": user,
	})
}

// ==============================
// LOGIN
// ==============================
func (h *AuthHandler) Login(c *gin.Context) {

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, accessToken, refreshToken, err :=
		h.service.Login(c.Request.Context(), req.Email, req.Password)

	if err != nil {

		switch err {
		case utils.ErrInvalidCredentials:
			utils.Error(c, http.StatusUnauthorized, "invalid credentials")
			return

		case utils.ErrUnauthorized:
			utils.Error(c, http.StatusForbidden, "account inactive")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	utils.Success(c, gin.H{
		"user":         user,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// ==============================
// REFRESH TOKEN
// ==============================
func (h *AuthHandler) Refresh(c *gin.Context) {

	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "refresh token required")
		return
	}

	newAccessToken, err :=
		h.service.Refresh(c.Request.Context(), req.RefreshToken)

	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	utils.Success(c, gin.H{
		"accessToken": newAccessToken,
	})
}

// ==============================
// LOGOUT (PROTECTED)
// ==============================
func (h *AuthHandler) Logout(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.service.Logout(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message": "logged out successfully",
	})
}

// ==============================
// CURRENT USER (PROTECTED)
// ==============================
func (h *AuthHandler) Me(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetUserByID(
		c.Request.Context(),
		userID.(string),
	)
	if err != nil {

		switch err {
		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "user not found")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	utils.Success(c, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name       string  `json:"name"`
		Department *string `json:"department"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(
		c.Request.Context(),
		userID.(string),
		req.Name,
		req.Department,
	)

	if err != nil {

		switch err {
		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "user not found")
			return
		default:
			utils.Error(c, http.StatusInternalServerError, "failed to update profile")
			return
		}
	}

	utils.Success(c, gin.H{"user": user})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.ChangePassword(
		c.Request.Context(),
		userID.(string),
		req.OldPassword,
		req.NewPassword,
	)

	if err != nil {

		switch err {
		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "user not found")
			return

		case utils.ErrInvalidCredentials:
			utils.Error(c, http.StatusUnauthorized, "invalid password")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, "failed to change password")
			return
		}
	}

	utils.Success(c, gin.H{"message": "password updated successfully"})
}

func (h *AuthHandler) GetAllUsers(c *gin.Context) {

	role, exists := c.Get("role")
	if !exists || role != "admin" {
		utils.Error(c, http.StatusForbidden, "forbidden")
		return
	}

	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	utils.Success(c, gin.H{"users": users})
}
