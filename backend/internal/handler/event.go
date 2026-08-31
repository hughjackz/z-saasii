package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/repository"
)

// GET /api/events/logs?date=YYYY-MM-DD&tenant_id=X&level=info
func GetEventLogs(c *gin.Context) {
	date := c.Query("date")
	level := c.Query("level")
	tenantID := c.Query("tenant_id")
	events, err := repository.QueryEvents(tenantID, date, level, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}
