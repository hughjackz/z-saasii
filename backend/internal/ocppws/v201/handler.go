package v201

// handler.go — OCPP 2.0.1 message dispatch (README 4.3).
// Payload field naming is CamelCase per the schemas in backend/doc/ocpp2.0.1/schema/.

import (
	"log"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
)

// HandleCall dispatches an incoming OCPP 2.0.1 CALL from a charge point.
// This function is called by the OCPP WebSocket server from the readPump.
func HandleCall(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	switch call.Action {
	case "Authorize":
		handleAuthorize(dc, call, eventCh)
	case "BootNotification":
		handleBootNotification(dc, call, eventCh)
	case "Heartbeat":
		handleHeartbeat(dc, call, eventCh)
	case "StatusNotification":
		handleStatusNotification(dc, call, eventCh)
	case "TransactionEvent":
		handleTransactionEvent(dc, call, eventCh)
	case "NotifyReport":
		sendResult(dc, call.MsgID, struct{}{})
	case "NotifyEvent":
		sendResult(dc, call.MsgID, struct{}{})
	case "CertificateSigned":
		handleCertificateSigned(dc, call, eventCh)
	case "DeleteCertificate":
		handleDeleteCertificate(dc, call, eventCh)
	case "Get15118EVCertificate":
		handleGet15118EVCertificate(dc, call, eventCh)
	case "GetInstalledCertificateIds":
		handleGetInstalledCertificateIds(dc, call, eventCh)
	case "InstallCertificate":
		handleInstallCertificate(dc, call, eventCh)
	case "SignCertificate":
		handleSignCertificate(dc, call, eventCh)
	default:
		log.Printf("[ocppws/v201] unhandled action %q from %s", call.Action, dc.DeviceName)
	}
}

// sendResult wraps a CALLRESULT for a standard OCPP action.
func sendResult(dc *ocppws.DeviceConnection, msgID string, payload interface{}) {
	msg, _ := ocppws.BuildCallResult(msgID, payload)
	dc.WriteCh <- msg
}

func init() {
	ocppws.V201Handler = HandleCall
}
