package routes

import (
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

	// PUBLIC
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	// PROTECTED
	authProtected := auth.Group("/")
	authProtected.Use(middleware.AuthMiddleware())

	authProtected.POST("/logout", authHandler.Logout)
	authProtected.GET("/me", authHandler.Me)

	// CHANGE PASSWORD belongs to auth
	authProtected.PUT("/change-password", authHandler.ChangePassword)

	// =========================
	// USER ROUTES
	// =========================
	users := router.Group("/users")
	users.Use(middleware.AuthMiddleware())

	// viewer endpoints
	users.PUT("/me", authHandler.UpdateProfile)

	// admin endpoints
	users.GET("/", authHandler.GetAllUsers)
}
