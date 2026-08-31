package ocpp

// helpers.go — shared helpers for OCPP HTTP handlers.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/middleware"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
)

// sendToOCPP forwards a request to the OCPP WebSocket server.
func sendToOCPP(c *gin.Context, deviceID, action string, payload interface{}) {
	if ocppws.Default == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OCPP server not ready"})
		return
	}
	result, err := ocppws.Default.SendRequest(deviceID, action, payload)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// tenantInfo extracts tenant scoping info from the Gin context (set by JWT middleware).
func tenantInfo(c *gin.Context) (callerRole model.Role, callerID string, tenantID string) {
	callerRole = model.Role(c.GetString(middleware.CtxRole))
	callerID = c.GetString(middleware.CtxUserID)
	tenantID = c.GetString(middleware.CtxTenantID)
	if callerRole == model.RoleCSAdmin {
		if qv := c.Query("tenant_id"); qv != "" {
			tenantID = qv
		}
	} else if tenantID == "" {
		tenantID = callerID
	}
	return
}
