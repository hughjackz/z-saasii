package v16

import (
	"encoding/json"
	"log"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
)

// HandleCall dispatches an incoming OCPP 1.6 CALL from a charge point.
// This function is called by the OCPP WebSocket server from the readPump.
func HandleCall(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	switch call.Action {
	case "BootNotification":
		handleBootNotification(dc, call, eventCh)
	case "Heartbeat":
		handleHeartbeat(dc, call, eventCh)
	case "StartTransaction":
		handleStartTransaction(dc, call, eventCh)
	case "StopTransaction":
		handleStopTransaction(dc, call, eventCh)
	case "MeterValues":
		handleMeterValues(dc, call, eventCh)
	case "Authorize":
		handleAuthorize(dc, call, eventCh)
	case "StatusNotification":
		handleStatusNotification(dc, call, eventCh)
	case "DiagnosticsStatusNotification":
		sendResult(dc, call.MsgID, struct{}{})
	case "FirmwareStatusNotification":
		sendResult(dc, call.MsgID, struct{}{})
	case "SecurityEventNotification":
		sendResult(dc, call.MsgID, struct{}{})
	case "LogStatusNotification":
		sendResult(dc, call.MsgID, struct{}{})
	case "SignCertificate":
		// PNC uses DataTransfer — if a standard SignCertificate arrives, still accept
		sendResult(dc, call.MsgID, map[string]string{"status": "Accepted"})
	case "DataTransfer":
		handleDataTransfer(dc, call, eventCh)
	default:
		log.Printf("[ocppws/v16] unhandled action %q from %s", call.Action, dc.DeviceName)
	}
}

// handleDataTransfer routes DataTransfer messages. Per OCPP 1.6, the payload is:
//
//	{vendorId: string, messageId?: string, data?: string}
//
// PNC messages use vendorId="org.openchargealliance.iso15118pnc"
// with the inner payload JSON-stringified in the data field.
type dataTransferPayload struct {
	VendorID  string `json:"vendorId"`
	MessageID string `json:"messageId"`
	Data      string `json:"data"` // JSON-stringified inner payload per OCPP 1.6 spec
}

func handleDataTransfer(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var dt dataTransferPayload
	if err := json.Unmarshal(call.Payload, &dt); err != nil {
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		return
	}
	if dt.VendorID != pncVendorId {
		sendDataTransferResult(dc, call.MsgID, "UnknownVendorId", "")
		return
	}

	// The inner PNC payload is JSON-stringified in the data field
	var innerPayload json.RawMessage
	if dt.Data != "" {
		innerPayload = json.RawMessage(dt.Data)
	}

	// Route to PNC handler based on messageId (see README 2.2)
	switch dt.MessageID {
	case "SignCertificate":
		call.Payload = innerPayload
		handlePNCSignCertificate(dc, call, eventCh)
	case "CertificateSigned":
		call.Payload = innerPayload
		handlePNCCertificateSigned(dc, call, eventCh)
	case "Get15118EVCertificate":
		call.Payload = innerPayload
		handlePNCGet15118EVCert(dc, call, eventCh)
	case "Authorize":
		call.Payload = innerPayload
		handlePNCAuthorize(dc, call, eventCh)
	case "GetInstalledCertificateIds":
		call.Payload = innerPayload
		handlePNCGetInstalledCertIds(dc, call, eventCh)
	case "InstallCertificate":
		call.Payload = innerPayload
		handlePNCInstallCertificate(dc, call, eventCh)
	case "DeleteCertificate":
		call.Payload = innerPayload
		handlePNCDeleteCertificate(dc, call, eventCh)
	case "TriggerMessage":
		call.Payload = innerPayload
		handlePNCTriggerMessage(dc, call, eventCh)
	case "GetCertificateStatus":
		call.Payload = innerPayload
		handlePNCGetCertStatus(dc, call, eventCh)
	default:
		log.Printf("[ocppws/v16] unhandled PNC DataTransfer messageId %q", dt.MessageID)
		sendDataTransferResult(dc, call.MsgID, "UnknownMessageId", "")
	}
}

// sendResult wraps a CALLRESULT for a standard OCPP action.
func sendResult(dc *ocppws.DeviceConnection, msgID string, payload interface{}) {
	msg, _ := ocppws.BuildCallResult(msgID, payload)
	dc.WriteCh <- msg
}

// sendDataTransferResult wraps a DataTransfer CALLRESULT with status and optional data.
func sendDataTransferResult(dc *ocppws.DeviceConnection, msgID, status, data string) {
	resp := map[string]string{"status": status}
	if data != "" {
		resp["data"] = data
	}
	sendResult(dc, msgID, resp)
}

// sendDataTransferError sends a DataTransfer CALLERROR.
func sendDataTransferError(dc *ocppws.DeviceConnection, msgID, data string) {
	sendError(dc, msgID, "FormationViolation", data)
}

func sendError(dc *ocppws.DeviceConnection, msgID, code, desc string) {
	msg, _ := ocppws.BuildCallError(msgID, code, desc, nil)
	dc.WriteCh <- msg
}
