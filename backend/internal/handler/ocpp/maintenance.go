package ocpp

// maintenance.go — Maintenance HTTP handlers (Reset, Firmware, GetLog).

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /api/ocpp/:deviceId/reset
func Reset(c *gin.Context) {
	var body struct {
		Type string `json:"type"` // Hard | Soft
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Type = "Hard"
	}
	sendToOCPP(c, c.Param("deviceId"), "Reset", map[string]string{"type": body.Type})
}

// POST /api/ocpp/:deviceId/get-log
func GetLog(c *gin.Context) {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	sendToOCPP(c, c.Param("deviceId"), "GetLog", body)
}

// POST /api/ocpp/:deviceId/firmware-update
func FirmwareUpdate(c *gin.Context) {
	file, err := c.FormFile("firmware")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "firmware file required"})
		return
	}
	dst := "/tmp/" + file.Filename
	_ = c.SaveUploadedFile(file, dst)

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	location := scheme + "://" + c.Request.Host + "/firmware/" + file.Filename

	sendToOCPP(c, c.Param("deviceId"), "UpdateFirmware", map[string]interface{}{
		"location":      location,
		"retrieveDate":  "now",
		"retries":       3,
		"retryInterval": 60,
	})
}
