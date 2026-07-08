package routes

import (
	"time"

	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication routes
func RegisterAuthRoutes(
	router *gin.RouterGroup,
	authHandler *handler.AuthHandler,
) {

	// =========================
	// AUTH ROUTES
	// =========================
	auth := router.Group("/auth")

	// 6-digit-code-guessing surfaces get a tighter limiter than the
	// general-purpose one (both are IP-based, in-memory, per-instance).
	codeGuessLimiter := middleware.NewRateLimiter(8, time.Minute)

	// PUBLIC
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/login/verify", codeGuessLimiter, authHandler.LoginVerify)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/reset-password", codeGuessLimiter, authHandler.ResetPasswordTwoFactor)

	// PROTECTED
	authProtected := auth.Group("/")
	authProtected.Use(middleware.AuthMiddleware())

	authProtected.POST("/logout", authHandler.Logout)
	authProtected.GET("/me", authHandler.Me)
	authProtected.PUT("/change-password", authHandler.ChangePassword)

	authProtected.POST("/2fa/enable", authHandler.Enable2FA)
	authProtected.POST("/2fa/verify", authHandler.Verify2FA)

	// =========================
	// USER ROUTES
	// =========================
	users := router.Group("/users")
	users.Use(middleware.AuthMiddleware())

	// viewer endpoints
	users.PUT("/me", authHandler.UpdateProfile)

	users.GET("/preferences", authHandler.GetPreferences)
	users.PUT("/preferences", authHandler.UpdatePreferences)
	users.GET("/stats/departments", authHandler.GetDepartmentStats)
	users.GET("/directory", authHandler.GetDirectory) // workspace member list — any auth'd user

	// user management endpoints — admin only
	users.GET("", middleware.AdminOnly(), authHandler.GetAllUsers)
	users.POST("", middleware.AdminOnly(), authHandler.AdminCreateUser)
	users.PATCH("/:id", middleware.AdminOnly(), authHandler.AdminUpdateUser)
	users.PATCH("/:id/status", middleware.AdminOnly(), authHandler.AdminToggleStatus)
	users.POST("/:id/reset-password", middleware.AdminOnly(), authHandler.AdminResetPassword)
	users.DELETE("/:id", middleware.AdminOnly(), authHandler.AdminDeleteUser)
}
