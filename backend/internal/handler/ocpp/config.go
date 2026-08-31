package ocpp

// config.go — GetConfiguration / ChangeConfiguration handlers.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/ocppws"
)

// GET /api/ocpp/:deviceId/configuration
func GetConfiguration(c *gin.Context) {
	sendToOCPP(c, c.Param("deviceId"), "GetConfiguration", map[string]interface{}{
		"key": []string{},
	})
}

// POST /api/ocpp/:deviceId/configuration/get
func GetConfigurationKeys(c *gin.Context) {
	var body struct {
		Keys []string `json:"keys"`
	}
	_ = c.ShouldBindJSON(&body)
	sendToOCPP(c, c.Param("deviceId"), "GetConfiguration", map[string]interface{}{
		"key": body.Keys,
	})
}

// POST /api/ocpp/:deviceId/configuration/set
func SetConfiguration(c *gin.Context) {
	var body struct {
		Configs []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"configs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deviceID := c.Param("deviceId")
	results := make([]map[string]interface{}, 0)
	for _, cfg := range body.Configs {
		resp, err := ocppws.Default.SendRequest(deviceID, "ChangeConfiguration",
			map[string]string{"key": cfg.Key, "value": cfg.Value})
		if err == nil {
			results = append(results, map[string]interface{}{
				"key":    cfg.Key,
				"status": resp,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
