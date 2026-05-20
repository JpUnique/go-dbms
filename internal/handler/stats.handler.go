package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	service *service.StatsService
}

func NewStatsHandler(service *service.StatsService) *StatsHandler {
	return &StatsHandler{service: service}
}

func (h *StatsHandler) Dashboard(c *gin.Context) {

	userID, _ := c.Get("userId")
	role, _ := c.Get("role")

	data, err := h.service.GetDashboard(
		c.Request.Context(),
		userID.(string),
		role.(string),
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	utils.Success(c, data)
}
func (h *StatsHandler) Activity(c *gin.Context) {

	userID, _ := c.Get("userId")
	role, _ := c.Get("role")

	data, err := h.service.GetActivity(
		c.Request.Context(),
		userID.(string),
		role.(string),
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to load activity")
		return
	}

	utils.Success(c, gin.H{"activity": data})
}
