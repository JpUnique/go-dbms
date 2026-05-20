package main

import (
	"context"
	"log"
	"net/http"

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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
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
	documentRepo := repository.NewDocumentRepository(dbConn)
	versionRepo := repository.NewDocumentVersionRepository(dbConn)
	folderRepo := repository.NewFolderRepository(dbConn)
	shareRepo := repository.NewShareRepository(dbConn)
	tagRepo := repository.NewTagRepository(dbConn)
	statsRepo := repository.NewStatsRepository(dbConn)
	bulkRepo := repository.NewBulkRepository(dbConn)
	trashRepo := repository.NewTrashRepository(dbConn)
	auditRepo := repository.NewAuditRepository(dbConn)
	// =========================
	// SERVICES
	// =========================
	authService := service.NewAuthService(userRepo, refreshRepo)
	documentService := service.NewDocumentService(documentRepo)
	versionService := service.NewDocumentVersionService(versionRepo, documentRepo)
	folderService := service.NewFolderService(folderRepo)
	shareService := service.NewShareService(shareRepo, documentRepo)
	tagService := service.NewTagService(tagRepo, documentRepo)
	statsService := service.NewStatsService(statsRepo)
	bulkService := service.NewBulkService(bulkRepo, folderRepo)
	trashService := service.NewTrashService(trashRepo)
	auditService := service.NewAuditService(auditRepo)

	// =========================
	// HANDLERS
	// =========================
	authHandler := handler.NewAuthHandler(authService)
	documentHandler := handler.NewDocumentHandler(documentService)
	versionHandler := handler.NewDocumentVersionHandler(versionService)
	folderHandler := handler.NewFolderHandler(folderService)
	shareHandler := handler.NewShareHandler(shareService)
	tagHandler := handler.NewTagHandler(tagService)
	statsHandler := handler.NewStatsHandler(statsService)
	bulkHandler := handler.NewBulkHandler(bulkService)
	trashHandler := handler.NewTrashHandler(trashService)
	auditHandler := handler.NewAuditHandler(auditService)

	// =========================
	// ROUTER
	// =========================
	router := gin.Default()

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

	// =========================
	// START SERVER
	// =========================
	log.Println("Server running on port:", cfg.Port)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
