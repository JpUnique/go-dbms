package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	service *service.ReportService
}

func NewReportHandler(service *service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

// Get — GET /reports?period=today|yesterday|week|month&scope=own|all
// (defaults to today, own). Every user generates their own report by
// default; scope=all is the system-wide, all-users view.
//
// TEMP-NO-ROLES: scope=all should be admin-only — restore the block below
// once role-based access is reintroduced.
func (h *ReportHandler) Get(c *gin.Context) {

	userID, _ := c.Get("userId")

	// TEMP-NO-ROLES: was
	// role, _ := c.Get("role")
	// if c.Query("scope") == "all" && role.(string) != "admin" {
	// 	utils.Error(c, http.StatusForbidden, "admin only")
	// 	return
	// }

	period := c.Query("period")
	allUsers := c.Query("scope") == "all"

	report, err := h.service.GetReport(c.Request.Context(), period, userID.(string), allUsers)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, report)
}
