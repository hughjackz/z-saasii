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

// idtagFieldMap maps accepted client field names to idtag table columns.
var idtagFieldMap = map[string]string{
	"tagId":       "tag_id",
	"tag_id":      "tag_id",
	"parentTagId": "parent_tag_id",
	"parent_tag_id": "parent_tag_id",
	"type":        "type",
	"status":      "status",
	"expiryTime":  "expiry_time",
	"expiry_time": "expiry_time",
}

func UpdateIDTag(c *gin.Context) {
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	normalized := make(map[string]interface{})
	for k, v := range fields {
		if col, ok := idtagFieldMap[k]; ok {
			normalized[col] = v
		}
	}
	if err := repository.UpdateIDTag(c.Param("id"), normalized); err != nil {
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
