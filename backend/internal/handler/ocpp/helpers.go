package ocpp

// helpers.go — shared helpers for OCPP HTTP handlers.

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/middleware"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
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

// isOcpp2Protocol reports whether a device speaks OCPP 2.x.
func isOcpp2Protocol(protocol string) bool {
	return strings.HasPrefix(strings.ToUpper(protocol), "OCPP2")
}

// deviceIsOcpp2 looks up the device protocol from the DB.
func deviceIsOcpp2(deviceID string) bool {
	d, err := repository.GetDevice(deviceID)
	if err != nil {
		return false
	}
	return isOcpp2Protocol(d.Protocol)
}

// actionMap201 translates OCPP 1.6 action names to their OCPP 2.0.1
// equivalents (README 4.3.15/4.3.16).
var actionMap201 = map[string]string{
	"GetConfiguration":       "GetVariables",
	"ChangeConfiguration":    "SetVariables",
	"RemoteStartTransaction": "RequestStartTransaction",
	"RemoteStopTransaction":  "RequestStopTransaction",
}

// sendDeviceAction sends an action to the device, translating action names
// when the device is OCPP 2.x.
func sendDeviceAction(c *gin.Context, deviceID, action string, payload interface{}) {
	if deviceIsOcpp2(deviceID) {
		if a, ok := actionMap201[action]; ok {
			action = a
		}
	}
	sendToOCPP(c, deviceID, action, payload)
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
