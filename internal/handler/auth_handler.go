package handler

import (
	"context"
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	service *service.AuthService
	db      *pgxpool.Pool
}

// constructor
func NewAuthHandler(service *service.AuthService, db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{
		service: service,
		db:      db,
	}
}

// logAuditByID records an auth event for a user that isn't necessarily
// authenticated yet on this request (e.g. login, register) — unlike
// DocumentHandler's logAudit, the user id comes from the service call
// result, not the gin context.
func (h *AuthHandler) logAuditByID(c *gin.Context, userID, action string) {
	if h.db == nil || userID == "" {
		return
	}
	ip, ua := c.ClientIP(), c.GetHeader("User-Agent")
	uid := userID
	go utils.LogAudit(context.Background(), h.db, utils.AuditEntry{
		UserID:       &uid,
		Action:       action,
		ResourceType: "auth",
		ResourceID:   &uid,
		IPAddress:    &ip,
		UserAgent:    &ua,
	})
}

// ==============================
// REGISTER
// ==============================
// Creates the account and starts mandatory 2FA setup. No tokens are issued
// until the caller completes POST /auth/login/verify with the returned
// challenge.
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

	user, challenge, qrCode, err := h.service.Register(
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

	h.logAuditByID(c, user.ID, "register")

	utils.Created(c, gin.H{
		"user":            user,
		"login_challenge": challenge,
		"qr_code":         qrCode,
		"status":          "2fa_setup_required",
	})
}

// ==============================
// LOGIN (step 1 — password check)
// ==============================
func (h *AuthHandler) Login(c *gin.Context) {

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	status, challenge, qrCode, user, err :=
		h.service.Login(c.Request.Context(), req.Username, req.Password)

	if err != nil {

		switch err {
		case utils.ErrInvalidCredentials:
			utils.Error(c, http.StatusUnauthorized, "invalid credentials")
			return

		case utils.ErrUnauthorized:
			utils.Error(c, http.StatusForbidden, "account is not active")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if user != nil {
		h.logAuditByID(c, user.ID, "login_password_verified")
	}

	utils.Success(c, gin.H{
		"status":          status,
		"login_challenge": challenge,
		"qr_code":         qrCode,
	})
}

// ==============================
// LOGIN VERIFY (step 2 — code check, issues tokens)
// ==============================
func (h *AuthHandler) LoginVerify(c *gin.Context) {

	var req struct {
		LoginChallenge string `json:"login_challenge" binding:"required"`
		Code           string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, accessToken, refreshToken, recoveryCodes, err :=
		h.service.LoginVerify(c.Request.Context(), req.LoginChallenge, req.Code)

	if err != nil {

		switch err {
		case utils.ErrUnauthorized:
			utils.Error(c, http.StatusUnauthorized, "login challenge expired — please sign in again")
			return

		case utils.ErrInvalidCredentials:
			utils.Error(c, http.StatusUnauthorized, "invalid code")
			return

		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h.logAuditByID(c, user.ID, "login")

	resp := gin.H{
		"user":         user,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}
	if len(recoveryCodes) > 0 {
		resp["recovery_codes"] = recoveryCodes
	}

	utils.Success(c, resp)
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

	h.logAuditByID(c, userID.(string), "logout")

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

	if len(req.NewPassword) < 8 {
		utils.Error(c, http.StatusBadRequest, "password must be at least 8 characters")
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
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h.logAuditByID(c, userID.(string), "change_password")

	utils.Success(c, gin.H{"message": "password updated successfully"})
}

func (h *AuthHandler) GetAllUsers(c *gin.Context) {

	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	utils.Success(c, gin.H{"users": users})
}

// GetDirectory returns safe public fields for ALL workspace members (any authenticated user).
func (h *AuthHandler) GetDirectory(c *gin.Context) {

	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch directory")
		return
	}

	type DirectoryUser struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Email      string  `json:"email"`
		Role       string  `json:"role"`
		Department *string `json:"department,omitempty"`
		CreatedAt  string  `json:"created_at"`
	}

	result := make([]DirectoryUser, 0, len(users))
	for _, u := range users {
		result = append(result, DirectoryUser{
			ID:         u.ID,
			Name:       u.Name,
			Email:      u.Email,
			Role:       u.Role,
			Department: u.Department,
			CreatedAt:  u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	utils.Success(c, gin.H{"users": result})
}

// ==============================
// RESET PASSWORD (TOTP / recovery code — no email involved)
// ==============================
func (h *AuthHandler) ResetPasswordTwoFactor(c *gin.Context) {

	var req struct {
		Username    string `json:"username" binding:"required"`
		Code        string `json:"code" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.NewPassword) < 8 {
		utils.Error(c, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	err := h.service.ResetPasswordViaTwoFactor(c.Request.Context(), req.Username, req.Code, req.NewPassword)
	if err != nil {
		// Always the same generic message — never reveal whether the
		// username, missing 2FA, or wrong code was the cause.
		utils.Error(c, http.StatusUnauthorized, "invalid username or code")
		return
	}

	utils.Success(c, gin.H{"message": "password reset successful"})
}

func (h *AuthHandler) Enable2FA(c *gin.Context) {

	userID, _ := c.Get("userId")

	qrURL, err := h.service.Enable2FA(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		utils.Error(c, 500, "failed to enable 2FA")
		return
	}

	utils.Success(c, gin.H{
		"qr_code": qrURL,
	})
}

func (h *AuthHandler) Verify2FA(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "invalid request")
		return
	}

	recoveryCodes, err := h.service.Verify2FA(
		c.Request.Context(),
		userID.(string),
		req.Code,
	)

	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	h.logAuditByID(c, userID.(string), "2fa_setup")

	utils.Success(c, gin.H{
		"message":        "2FA verified",
		"recovery_codes": recoveryCodes,
	})
}

func (h *AuthHandler) GetPreferences(c *gin.Context) {

	userID, _ := c.Get("userId")

	prefs, err := h.service.GetPreferences(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		utils.Error(c, 500, "failed to get preferences")
		return
	}

	utils.Success(c, gin.H{"preferences": prefs})
}

func (h *AuthHandler) UpdatePreferences(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		DarkMode           bool `json:"dark_mode"`
		EmailNotifications bool `json:"email_notifications"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "invalid request")
		return
	}

	err := h.service.UpdatePreferences(
		c.Request.Context(),
		userID.(string),
		req.DarkMode,
		req.EmailNotifications,
	)

	if err != nil {
		utils.Error(c, 500, "failed to update preferences")
		return
	}

	utils.Success(c, gin.H{"message": "preferences updated"})
}

func (h *AuthHandler) GetDepartmentStats(c *gin.Context) {
	stats, err := h.service.GetDepartmentStats(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch department stats")
		return
	}
	utils.Success(c, gin.H{"departments": stats})
}

// AdminCreateUser — POST /users
func (h *AuthHandler) AdminCreateUser(c *gin.Context) {
	var req struct {
		Name       string  `json:"name" binding:"required"`
		Email      string  `json:"email" binding:"required,email"`
		Password   string  `json:"password" binding:"required"`
		Role       string  `json:"role"`
		Department *string `json:"department"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.service.AdminCreateUser(c.Request.Context(), req.Email, req.Password, req.Name, req.Role, req.Department)
	if err != nil {
		utils.Error(c, http.StatusConflict, "email already in use")
		return
	}
	utils.Created(c, gin.H{"user": user})
}

// AdminUpdateUser — PATCH /users/:id
func (h *AuthHandler) AdminUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Name       string  `json:"name"`
		Email      string  `json:"email"`
		Role       string  `json:"role"`
		Department *string `json:"department"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.service.AdminUpdateUser(c.Request.Context(), userID, req.Name, req.Email, req.Role, req.Department)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to update user")
		return
	}
	utils.Success(c, gin.H{"user": user})
}

// AdminToggleStatus — PATCH /users/:id/status
func (h *AuthHandler) AdminToggleStatus(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "status required")
		return
	}
	if req.Status != "active" && req.Status != "inactive" {
		utils.Error(c, http.StatusBadRequest, "status must be active or inactive")
		return
	}
	user, err := h.service.ToggleUserStatus(c.Request.Context(), userID, req.Status)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to update status")
		return
	}
	utils.Success(c, gin.H{"user": user})
}

// AdminResetPassword — POST /users/:id/reset-password
func (h *AuthHandler) AdminResetPassword(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "password required")
		return
	}
	if err := h.service.AdminResetPassword(c.Request.Context(), userID, req.Password); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to reset password")
		return
	}
	utils.Success(c, gin.H{"message": "password reset successfully"})
}

// AdminDeleteUser — DELETE /users/:id
func (h *AuthHandler) AdminDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if err := h.service.AdminDeleteUser(c.Request.Context(), userID); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to delete user")
		return
	}
	utils.Success(c, gin.H{"message": "user deleted"})
}
