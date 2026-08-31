package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

func ListIDTags(c *gin.Context) {
	callerRole, callerID, tenantID := tenantDB(c)
	tags, err := repository.ListIDTags(callerRole, callerID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tags)
}

func CreateIDTag(c *gin.Context) {
	_, _, tenantID := tenantDB(c)

	var t model.IDTag
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.TenantID == "" { t.TenantID = tenantID }
	t.OwnerID = t.TenantID
	if err := repository.CreateIDTag(&t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func UpdateIDTag(c *gin.Context) {
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, f := range []string{
		"id",
		"createdAt", "created_at",
		"updatedAt", "updated_at",
		"ownerId", "owner_id",
		"tenantId", "tenant_id",
	} {
		delete(fields, f)
	}
	if err := repository.UpdateIDTag(c.Param("id"), fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteIDTag(c *gin.Context) {
	if err := repository.DeleteIDTag(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
