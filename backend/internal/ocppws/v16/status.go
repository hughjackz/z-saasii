package v16

// status.go — Device status notification handler.
// Handles StatusNotification from charge points.

import (
	"encoding/json"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── StatusNotification ─────────────────────────────────────────────────────
// Schema: StatusNotification.json / StatusNotificationResponse.json (empty)

type statusNotificationReq struct {
	ConnectorID     int    `json:"connectorId"`
	ErrorCode       string `json:"errorCode"`
	Info            string `json:"info"`
	Status          string `json:"status"`
	Timestamp       string `json:"timestamp"`
	VendorID        string `json:"vendorId"`
	VendorErrorCode string `json:"vendorErrorCode"`
}

func handleStatusNotification(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req statusNotificationReq
	_ = json.Unmarshal(call.Payload, &req)

	_ = repository.UpdateDeviceStatus(dc.DeviceName, req.Status)

	sendResult(dc, call.MsgID, struct{}{})

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "StatusNotification: "+req.Status+" (connector="+itoa(req.ConnectorID)+")")
}
