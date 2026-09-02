package ocpp

// tx.go — Transaction-related HTTP handlers.
// Protocol-aware (README 4.3.16): OCPP 1.6 uses RemoteStart/RemoteStop
// Transaction; OCPP 2.0.1 uses RequestStartTransaction/RequestStopTransaction
// and its own transaction_201 table.

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/repository"
)

// GET /api/ocpp/:deviceId/transactions/active
func GetActiveTransactions(c *gin.Context) {
	callerRole, callerID, tenantID := tenantInfo(c)
	deviceID := c.Param("deviceId")
	if deviceIsOcpp2(deviceID) {
		txs, err := repository.ListActiveTransactions201(callerRole, callerID, tenantID, deviceNameOf(deviceID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, txs)
		return
	}
	txs, err := repository.ListActiveTransactions(callerRole, callerID, tenantID, deviceNameOf(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

// GET /api/ocpp/:deviceId/transactions
// Optional ?transaction_id= filter (README 2.3.2.2.1 transactionid 筛选器).
func GetTransactions(c *gin.Context) {
	callerRole, callerID, tenantID := tenantInfo(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	txIDFilter := c.Query("transaction_id")
	deviceID := c.Param("deviceId")
	deviceName := deviceNameOf(deviceID)

	if deviceIsOcpp2(deviceID) {
		result, err := repository.ListTransactions201(callerRole, callerID, tenantID, deviceName, txIDFilter, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	result, err := repository.ListTransactions(callerRole, callerID, tenantID, deviceName, txIDFilter, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/ocpp/:deviceId/transaction-events
// Recorded TransactionEvents for an OCPP 2.0.1 device. Meter-sampling events
// are excluded by default (?include_meter=1 to include).
func GetTransactionEvents(c *gin.Context) {
	_, _, tenantID := tenantInfo(c)
	deviceID := c.Param("deviceId")
	deviceName := deviceNameOf(deviceID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	includeMeter := c.Query("include_meter") == "1"
	events, err := repository.ListTransactionEvents201(deviceName, tenantID, includeMeter, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// POST /api/ocpp/:deviceId/remote-start
// Body: {connectorId, idTag, evseId? (OCPP201), profileId?}
func RemoteStart(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deviceID := c.Param("deviceId")

	if deviceIsOcpp2(deviceID) {
		// RequestStartTransaction: {remoteStartId, idToken, evseId?, chargingProfile?}
		payload := map[string]interface{}{
			"remoteStartId": 1,
			"idToken": map[string]interface{}{
				"idToken": body["idTag"],
				"type":    "ISO14443",
			},
		}
		if evseID, ok := body["evseId"].(float64); ok && evseID > 0 {
			payload["evseId"] = int(evseID)
		}
		if profileID, ok := body["profileId"].(string); ok && profileID != "" {
			payload["chargingProfile"] = map[string]interface{}{"id": profileID}
		}
		sendToOCPP(c, deviceID, "RequestStartTransaction", payload)
		return
	}

	sendToOCPP(c, deviceID, "RemoteStartTransaction", body)
}

// POST /api/ocpp/:deviceId/remote-stop
// Body: {transactionId (int for v16 / string for v201)}
func RemoteStop(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deviceID := c.Param("deviceId")

	if deviceIsOcpp2(deviceID) {
		// RequestStopTransaction: {transactionId (string)}
		txID := ""
		switch v := body["transactionId"].(type) {
		case string:
			txID = v
		case float64:
			txID = strconv.FormatInt(int64(v), 10)
		}
		sendToOCPP(c, deviceID, "RequestStopTransaction", map[string]string{
			"transactionId": txID,
		})
		return
	}

	sendToOCPP(c, deviceID, "RemoteStopTransaction", body)
}

// deviceNameOf resolves a device's name from its id (transactions are keyed by
// charge point id == device name).
func deviceNameOf(deviceID string) string {
	if d, err := repository.GetDevice(deviceID); err == nil {
		return d.Name
	}
	return deviceID
}
