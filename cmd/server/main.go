package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JpUnique/go-dbms/internal/config"
	"github.com/JpUnique/go-dbms/internal/db"
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/repository"
	routes "github.com/JpUnique/go-dbms/internal/routes"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// =========================
	// LOAD ENV
	// =========================

	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("../../.env")
	}

	if err != nil {
		log.Println("No env variables found")
	} else {
		log.Println(" The env variables are loaded successfully ✅")
	}

	// =========================
	// CONFIG
	// =========================
	cfg := config.LoadConfig()

	ctx := context.Background()

	// =========================
	// DATABASE
	// =========================
	dbConn := db.ConnectDB()
	defer db.CloseDB()

	if cfg.RunMigration {
		if err := db.RunMigration(ctx); err != nil {
			log.Fatal(err)
		}
	}

	if cfg.RunSeed {
		if err := db.RunSeed(ctx); err != nil {
			log.Fatal(err)
		}
	}

	// =========================
	// STORAGE (MinIO)
	// =========================
	if err := storage.InitMinIO(); err != nil {
		log.Fatal("failed to initialize MinIO:", err)
	}

	// =========================
	// REPOSITORIES
	// =========================
	userRepo := repository.NewUserRepository(dbConn)
	refreshRepo := repository.NewRefreshTokenRepository(dbConn)
	recoveryCodeRepo := repository.NewRecoveryCodeRepository(dbConn)
	documentRepo := repository.NewDocumentRepository(dbConn)
	versionRepo := repository.NewDocumentVersionRepository(dbConn)
	folderRepo := repository.NewFolderRepository(dbConn)
	shareRepo := repository.NewShareRepository(dbConn)
	tagRepo := repository.NewTagRepository(dbConn)
	statsRepo := repository.NewStatsRepository(dbConn)
	bulkRepo := repository.NewBulkRepository(dbConn)
	trashRepo := repository.NewTrashRepository(dbConn)
	auditRepo := repository.NewAuditRepository(dbConn)
	commentRepo := repository.NewCommentRepository(dbConn)
	notificationRepo := repository.NewNotificationRepository(dbConn)
	reviewRepo := repository.NewReviewRepository(dbConn)
	watcherRepo := repository.NewWatcherRepository(dbConn)
	ragRepo := repository.NewRAGRepository(dbConn)
	reportRepo := repository.NewReportRepository(dbConn)
	userShareRepo := repository.NewUserShareRepository(dbConn)
	// =========================
	// SERVICES
	// =========================
	authService := service.NewAuthService(userRepo, refreshRepo, recoveryCodeRepo)
	documentService := service.NewDocumentService(documentRepo, userRepo)
	versionService := service.NewDocumentVersionService(versionRepo, documentRepo, userRepo)
	folderService := service.NewFolderService(folderRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	shareService := service.NewShareService(shareRepo, documentRepo, notificationService)
	tagService := service.NewTagService(tagRepo, documentRepo, userRepo)
	statsService := service.NewStatsService(statsRepo)
	bulkService := service.NewBulkService(bulkRepo, folderRepo)
	trashService := service.NewTrashService(trashRepo)
	auditService := service.NewAuditService(auditRepo)
	commentService := service.NewCommentService(commentRepo, userRepo)
	reviewService := service.NewReviewService(reviewRepo, documentRepo, userRepo)
	watcherService := service.NewWatcherService(watcherRepo)
	ragService := service.NewRAGService(ragRepo)
	reportService := service.NewReportService(reportRepo)
	userShareService := service.NewUserShareService(userShareRepo, documentRepo, notificationService, dbConn)

	// =========================
	// HANDLERS
	// =========================
	authHandler := handler.NewAuthHandler(authService, dbConn)
	documentHandler := handler.NewDocumentHandler(documentService, dbConn, notificationService, ragService)
	versionHandler := handler.NewDocumentVersionHandler(versionService)
	folderHandler := handler.NewFolderHandler(folderService)
	shareHandler := handler.NewShareHandler(shareService)
	tagHandler := handler.NewTagHandler(tagService)
	statsHandler := handler.NewStatsHandler(statsService)
	bulkHandler := handler.NewBulkHandler(bulkService)
	trashHandler := handler.NewTrashHandler(trashService)
	auditHandler := handler.NewAuditHandler(auditService)
	commentHandler := handler.NewCommentHandler(commentService, notificationService, watcherService, documentService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	reviewHandler := handler.NewReviewHandler(reviewService, notificationService, authService)
	watcherHandler := handler.NewWatcherHandler(watcherService)
	ragHandler := handler.NewRAGHandler(ragService)
	reportHandler := handler.NewReportHandler(reportService)
	userShareHandler := handler.NewUserShareHandler(userShareService)

	// =========================
	// ROUTER
	// =========================
	// Use gin.New() so we control middleware order explicitly:
	// CORS must be first so its headers are always present, even on panics.
	router := gin.New()
	router.RedirectTrailingSlash = false

	// =========================
	// CORS — handwritten, secure, and production-ready
	// =========================
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Define allowed origins explicitly
		allowedOrigins := map[string]bool{
			"http://localhost:3000":     true,
			"http://140.238.79.81:3000": true, // Oracle Cloud deployment
			"http://100.91.202.86:3000": true, // Physical server (Tailscale)
			// Add production domain here later: "https://yourdomain.com": true,
		}

		// Handle cases where Origin header is missing (like same-origin mobile apps/curl)
		if origin == "" {
			c.Next()
			return
		}

		// Reject unauthorized origins immediately during preflight
		if !allowedOrigins[origin] {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Inject headers safely for allowed origins
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Max-Age", "43200") // Cache preflight for 12 hours
		c.Header("Vary", "Origin")

		// Successfully terminate authorized OPTIONS preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Logger and Recovery AFTER CORS so every response has CORS headers
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Every API response here is dynamic, per-user data — instruct browsers
	// and any intermediary proxy to never cache it, so a client-side nav
	// back to a page always re-fetches current state instead of risking a
	// stale cached GET.
	router.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// =========================
	// VERSIONED BASE ROUTE
	// =========================
	base := "/dbms/" + cfg.API.Version
	app := router.Group(base)

	// =========================
	// REGISTER ROUTES
	// =========================
	routes.RegisterAuthRoutes(app, authHandler)
	routes.RegisterDocumentRoutes(app, documentHandler)
	routes.RegisterDocumentVersionRoutes(app, versionHandler)
	routes.RegisterFolderRoutes(app, folderHandler)
	routes.RegisterShareRoutes(app, shareHandler)
	routes.RegisterTagRoutes(app, tagHandler)
	routes.RegisterStatsRoutes(app, statsHandler)
	routes.RegisterBulkRoutes(app, bulkHandler)
	routes.RegisterTrashRoutes(app, trashHandler)
	routes.RegisterAuditRoutes(app, auditHandler)
	routes.RegisterCommentRoutes(app, commentHandler)
	routes.RegisterNotificationRoutes(app, notificationHandler)
	routes.RegisterReviewRoutes(app, reviewHandler)
	routes.RegisterWatcherRoutes(app, watcherHandler)
	routes.RegisterRAGRoutes(app, ragHandler)
	routes.RegisterReportRoutes(app, reportHandler)
	routes.RegisterUserShareRoutes(app, userShareHandler)

	// =========================
	// START SERVER
	// =========================
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Println("Server running on port:", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen error:", err)
		}
	}()

	// Wait for SIGINT or SIGTERM (Ctrl+C or kill).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down — waiting for in-flight requests (10s max)…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("forced shutdown:", err)
	}

	log.Println("Server stopped cleanly.")
}
