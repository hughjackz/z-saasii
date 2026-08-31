package ocpp

// tx.go — Transaction-related HTTP handlers.

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/repository"
)

// GET /api/ocpp/:deviceId/transactions/active
func GetActiveTransactions(c *gin.Context) {
	callerRole, callerID, tenantID := tenantInfo(c)
	txs, err := repository.ListActiveTransactions(callerRole, callerID, tenantID, c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

// GET /api/ocpp/:deviceId/transactions
func GetTransactions(c *gin.Context) {
	callerRole, callerID, tenantID := tenantInfo(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	result, err := repository.ListTransactions(callerRole, callerID, tenantID, c.Param("deviceId"), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/ocpp/:deviceId/remote-start
func RemoteStart(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sendToOCPP(c, c.Param("deviceId"), "RemoteStartTransaction", body)
}

// POST /api/ocpp/:deviceId/remote-stop
func RemoteStop(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sendToOCPP(c, c.Param("deviceId"), "RemoteStopTransaction", body)
}
