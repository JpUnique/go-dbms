package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type FolderHandler struct {
	service *service.FolderService
}

// constructor
func NewFolderHandler(service *service.FolderService) *FolderHandler {
	return &FolderHandler{service: service}
}

// ==============================
// GET ALL FOLDERS
// ==============================
func (h *FolderHandler) GetAll(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// query params
	parentID := c.Query("parent_id")

	limit := 200
	offset := 0

	folders, err := h.service.GetAllFolders(
		c.Request.Context(),
		userID.(string),
		parentID,
		limit,
		offset,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch folders")
		return
	}

	utils.Success(c, gin.H{
		"folders": folders,
	})
}

// ==============================
// GET SINGLE FOLDER
// ==============================
func (h *FolderHandler) GetOne(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	folderID := c.Param("id")

	folder, err := h.service.GetByID(
		c.Request.Context(),
		folderID,
		userID.(string),
	)

	if err != nil {

		switch err {
		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "folder not found")
		default:
			utils.Error(c, http.StatusInternalServerError, "failed to fetch folder")
		}
		return
	}

	utils.Success(c, gin.H{
		"folder": folder,
	})
}

// ==============================
// CREATE FOLDER
// ==============================
func (h *FolderHandler) Create(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name       string  `json:"name" binding:"required"`
		ParentID   *string `json:"parent_id"`
		Department *string `json:"department"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	folder, err := h.service.CreateFolder(
		c.Request.Context(),
		userID.(string),
		req.Name,
		req.ParentID,
		req.Department,
	)

	if err != nil {

		//  business error mapping
		if err.Error() == "invalid parent folder" {
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}

		utils.Error(c, http.StatusInternalServerError, "failed to create folder")
		return
	}

	utils.Created(c, gin.H{
		"folder": folder,
	})
}

// ==============================
// UPDATE FOLDER
// ==============================
func (h *FolderHandler) Update(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	folderID := c.Param("id")

	var req struct {
		Name       *string `json:"name"`
		ParentID   *string `json:"parent_id"`
		Department *string `json:"department"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	folder, err := h.service.UpdateFolder(
		c.Request.Context(),
		folderID,
		userID.(string),
		req.Name,
		req.ParentID,
		req.Department,
	)

	if err != nil {

		switch err {

		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "folder not found")

		case utils.ErrInvalidInput:
			utils.Error(c, http.StatusBadRequest, "invalid input")

		default:
			utils.Error(c, http.StatusInternalServerError, "failed to update folder")
		}

		return
	}

	utils.Success(c, gin.H{
		"folder": folder,
	})
}

// ==============================
// DELETE FOLDER
// ==============================
func (h *FolderHandler) Delete(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	folderID := c.Param("id")

	err := h.service.DeleteFolder(
		c.Request.Context(),
		folderID,
		userID.(string),
	)

	if err != nil {

		switch err {
		case utils.ErrNotFound:
			utils.Error(c, http.StatusNotFound, "folder not found")
		default:
			utils.Error(c, http.StatusInternalServerError, "failed to delete folder")
		}

		return
	}

	utils.Success(c, gin.H{
		"message": "folder deleted successfully",
	})
}
