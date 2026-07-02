package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type RAGHandler struct {
	svc *service.RAGService
}

func NewRAGHandler(svc *service.RAGService) *RAGHandler {
	return &RAGHandler{svc: svc}
}

// POST /chat/ask
func (h *RAGHandler) Ask(c *gin.Context) {
	userID, _ := c.Get("userId")

	var req service.AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Ask(c.Request.Context(), userID.(string), req)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{"result": resp})
}

// POST /chat/sessions
func (h *RAGHandler) CreateSession(c *gin.Context) {
	userID, _ := c.Get("userId")

	var body struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&body)

	sess, err := h.svc.CreateSession(c.Request.Context(), userID.(string), body.Title)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to create session")
		return
	}

	utils.Created(c, gin.H{"session": sess})
}

// GET /chat/sessions
func (h *RAGHandler) ListSessions(c *gin.Context) {
	userID, _ := c.Get("userId")

	sessions, err := h.svc.ListSessions(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	utils.Success(c, gin.H{"sessions": sessions})
}

// GET /chat/sessions/:id
func (h *RAGHandler) GetSession(c *gin.Context) {
	userID, _ := c.Get("userId")
	sessionID := c.Param("id")

	sess, msgs, err := h.svc.GetSession(c.Request.Context(), sessionID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "session not found")
		return
	}

	utils.Success(c, gin.H{"session": sess, "messages": msgs})
}

// DELETE /chat/sessions/:id
func (h *RAGHandler) DeleteSession(c *gin.Context) {
	userID, _ := c.Get("userId")
	sessionID := c.Param("id")

	if err := h.svc.DeleteSession(c.Request.Context(), sessionID, userID.(string)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to delete session")
		return
	}

	utils.Success(c, gin.H{"message": "session deleted"})
}
