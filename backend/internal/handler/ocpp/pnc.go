package ocpp

// pnc.go — Plug & Charge (PnC) HTTP handlers.
// Handles certificate operations on charge points via OCPP WebSocket.
// PNC uses DataTransfer with vendorId="org.openchargealliance.iso15118pnc"
// where the inner payload is JSON-stringified in the data field.

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/ocppws/v16"
	"github.com/yourorg/csms-backend/internal/repository"
)

const pncVendorID = "org.openchargealliance.iso15118pnc"

// sendPNCDataTransfer wraps a payload in a DataTransfer CALL and sends it to the device.
func sendPNCDataTransfer(c *gin.Context, deviceID, messageID string, innerPayload interface{}) {
	if innerPayload == nil {
		innerPayload = struct{}{}
	}
	dataBytes, _ := json.Marshal(innerPayload)
	wrapper := map[string]string{
		"vendorId":  pncVendorID,
		"messageId": messageID,
		"data":      string(dataBytes),
	}
	sendToOCPP(c, deviceID, "DataTransfer", wrapper)
}

// POST /api/ocpp/:deviceId/get-installed-certificate-ids
// Schema: certificateType is an array of enum strings.
func GetInstalledCertificateIds(c *gin.Context) {
	var req struct {
		CertType string `json:"certificateType"`
	}
	_ = c.ShouldBindJSON(&req)

	// Map display type to schema enum array
	var certTypes []string
	switch req.CertType {
	case "MO Root":
		certTypes = []string{"MORootCertificate"}
	case "V2G Root":
		certTypes = []string{"V2GRootCertificate"}
	case "":
		certTypes = []string{} // empty = all types
	default:
		certTypes = []string{req.CertType}
	}

	// Send via DataTransfer and unwrap the inner data for the frontend
	dataBytes, _ := json.Marshal(map[string]interface{}{"certificateType": certTypes})
	wrapper := map[string]string{
		"vendorId":  pncVendorID,
		"messageId": "GetInstalledCertificateIds",
		"data":      string(dataBytes),
	}
	result, err := ocppws.Default.SendRequest(c.Param("deviceId"), "DataTransfer", wrapper)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	// Unwrap DataTransfer response: {status, data} → extract data string → parse JSON
	if respMap, ok := result.(map[string]interface{}); ok {
		if dataStr, ok := respMap["data"].(string); ok && dataStr != "" {
			var inner interface{}
			if json.Unmarshal([]byte(dataStr), &inner) == nil {
				result = inner
			}
		}
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/ocpp/:deviceId/delete-certificate
// Looks up the certificate in the DB to get hash data, then sends a
// DataTransfer DeleteCertificate request to the device (per README 4.2.9.3).
func DeleteCertificateOnDevice(c *gin.Context) {
	var req struct {
		CertName string `json:"certName"`
	}
	_ = c.ShouldBindJSON(&req)

	deviceID := c.Param("deviceId")
	_, callerID, tenantID := tenantInfo(c)

	allCerts, err := repository.ListCertificates(model.RoleCSAdmin, callerID, tenantID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find the certificate and extract hash data
	var certHashData map[string]string
	for _, cert := range allCerts {
		if cert.Name == req.CertName {
			certHashData = map[string]string{
				"hashAlgorithm":  cert.HashAlgorithm,
				"issuerNameHash": cert.IssuerNameHash,
				"issuerKeyHash":  cert.IssuerKeyHash,
				"serialNumber":   cert.SerialNumber,
			}
			break
		}
	}
	if certHashData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found: " + req.CertName})
		return
	}

	sendPNCDataTransfer(c, deviceID, "DeleteCertificate", map[string]interface{}{
		"certificateHashData": certHashData,
	})
}

// POST /api/ocpp/:deviceId/install-certificate
// Accepts a list of certificate names, looks up each one, and sends them
// one-by-one to the device in separate DataTransfer frames (per README 4.2.9.4).
func InstallCertificate(c *gin.Context) {
	var req struct {
		CertNames []string `json:"certNames"`
		CertType  string   `json:"certType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.CertNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "certNames required (list)"})
		return
	}
	if req.CertType == "" {
		req.CertType = "V2GRootCertificate"
	}

	_, callerID, tenantID := tenantInfo(c)
	deviceID := c.Param("deviceId")
	allCerts, err := repository.ListCertificates(model.RoleCSAdmin, callerID, tenantID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Install each certificate in its own frame
	results := make([]map[string]interface{}, 0, len(req.CertNames))
	for _, certName := range req.CertNames {
		var certID string
		for _, cert := range allCerts {
			if cert.Name == certName {
				certID = cert.ID
				break
			}
		}
		if certID == "" {
			results = append(results, map[string]interface{}{"name": certName, "status": "NotFound"})
			continue
		}
		content, _, err := repository.GetCertificateContent(certID)
		if err != nil {
			results = append(results, map[string]interface{}{"name": certName, "status": "Error"})
			continue
		}

		// Map display type to schema enum: "MO Root"→"MORootCertificate", "V2G Root"→"V2GRootCertificate"
		schemaType := req.CertType
		switch req.CertType {
		case "MO Root": schemaType = "MORootCertificate"
		case "V2G Root": schemaType = "V2GRootCertificate"
		}
		inner, _ := json.Marshal(map[string]string{"certificateType": schemaType, "certificate": content})
		wrapper := map[string]string{
			"vendorId":  pncVendorID,
			"messageId": "InstallCertificate",
			"data":      string(inner),
		}
		_, err = ocppws.Default.SendRequest(deviceID, "DataTransfer", wrapper)
		if err != nil {
			results = append(results, map[string]interface{}{"name": certName, "status": "Failed", "error": err.Error()})
		} else {
			results = append(results, map[string]interface{}{"name": certName, "status": "Sent"})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// POST /api/ocpp/:deviceId/trigger-message
func TriggerMessage(c *gin.Context) {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	sendPNCDataTransfer(c, c.Param("deviceId"), "TriggerMessage", body)
}

// POST /api/ocpp/:deviceId/sign-certificate
// Step 1 of SECC Leaf signing (4.2.9.1):
// Records the selected V2G cert chain, then triggers the device.
func SignCertificate(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var req struct {
		V2GRoot string `json:"v2gRoot"`
		V2GSub1 string `json:"v2gSub1"`
		V2GSub2 string `json:"v2gSub2"`
	}
	_ = c.ShouldBindJSON(&req)

	// Resolve CP_OP name for file paths
	device, err := repository.GetDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	op, _ := repository.GetUserByID(device.TenantID)
	cpopName := ""
	if op != nil {
		cpopName = op.Username
	}

	// Record the V2G cert chain selection for this device
	v16.RecordSECCSession(deviceID, req.V2GRoot, req.V2GSub1, req.V2GSub2, cpopName)

	// Trigger device to send SignCertificate request
	sendPNCDataTransfer(c, deviceID, "TriggerMessage", map[string]string{
		"requestedMessage": "SignCertificate",
	})
}

// POST /api/ocpp/:deviceId/certificate-signed
// Sends a signed certificate to the device (4.2.9.1 step 3).
func CertificateSigned(c *gin.Context) {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	sendPNCDataTransfer(c, c.Param("deviceId"), "CertificateSigned", body)
}

// POST /api/ocpp/:deviceId/get-certificate-status
func GetCertificateStatus(c *gin.Context) {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	sendPNCDataTransfer(c, c.Param("deviceId"), "GetCertificateStatus", body)
}

// POST /api/ocpp/:deviceId/contract-cert-group
// Saves the user-selected contract certificate group (2.3.2.4.e).
// When the device sends Get15118EVCertificate, the stored group is used by ContractGenerate.
func SaveContractCertGroup(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var req struct {
		Certs map[string]string `json:"certs"` // {"V2G-root-cert":"name1", "CPO-sub1-cert":"name2", ...}
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Certs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "certs map required"})
		return
	}

	_, _, tenantID := tenantInfo(c)
	v16.SetContractCertGroup(deviceID, tenantID, req.Certs)
	c.JSON(http.StatusOK, gin.H{"message": "contract cert group saved", "deviceId": deviceID})
}
