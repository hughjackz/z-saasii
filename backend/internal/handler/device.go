package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/middleware"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// tenantDB extracts tenant_id from context for scoped queries.
// For CS_Admin, tenant_id can be overridden via ?tenant_id= query param
// (the frontend TenantSelector passes the selected CP_OP's id).
func tenantDB(c *gin.Context) (callerRole model.Role, callerID string, tenantID string) {
	callerRole = model.Role(c.GetString(middleware.CtxRole))
	callerID = c.GetString(middleware.CtxUserID)
	tenantID = c.GetString(middleware.CtxTenantID)
	if callerRole == model.RoleCSAdmin {
		// CS_Admin: use query param to scope to a tenant, otherwise show all
		if qv := c.Query("tenant_id"); qv != "" {
			tenantID = qv
		}
	} else if tenantID == "" {
		tenantID = callerID // CP_OP/CP_OM: fallback to own id
	}
	return
}

func ListDevices(c *gin.Context) {
	callerRole, callerID, tenantID := tenantDB(c)

	devices, err := repository.ListDevices(callerRole, callerID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reflect real-time connection status from WebSocket hub (in-memory only)
	if ocppws.Default != nil {
		for _, d := range devices {
			d.Online = ocppws.Default.IsConnected(d.ID)
		}
	}

	c.JSON(http.StatusOK, devices)
}

func GetDeviceHandler(c *gin.Context) {
	d, err := repository.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func CreateDevice(c *gin.Context) {
	_, _, tenantID := tenantDB(c)

	var d model.Device
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// All roles: owner = tenant (CP_OP). tenant_id is set from JWT or request body.
	if d.TenantID == "" { d.TenantID = tenantID }
	d.OwnerID = &d.TenantID

	created, err := repository.CreateDevice(&d)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func UpdateDevice(c *gin.Context) {
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, f := range []string{
		"id", "ownerName",
		"createdAt", "created_at",
		"updatedAt", "updated_at",
		"lastHeartbeat", "last_heartbeat",
		"status",
		"tenantId", "tenant_id", // tenant cannot be changed
	} {
		delete(fields, f)
	}
	if err := repository.UpdateDevice(c.Param("id"), fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	d, _ := repository.GetDevice(c.Param("id"))
	c.JSON(http.StatusOK, d)
}

func DeleteDevice(c *gin.Context) {
	if err := repository.DeleteDevice(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
