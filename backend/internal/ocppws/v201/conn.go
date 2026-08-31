package v201

// conn.go — Device connection management handlers.
// Handles BootNotification, Heartbeat, and StatusNotification from charge points.

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── BootNotification ────────────────────────────────────────────────────────
// Schema: BootNotificationRequest.json / BootNotificationResponse.json
// Response requires currentTime, interval, status.

type bootNotificationReq struct {
	Reason          string `json:"reason"`
	ChargingStation struct {
		SerialNumber    string `json:"serialNumber"`
		Model           string `json:"model"`
		VendorName      string `json:"vendorName"`
		FirmwareVersion string `json:"firmwareVersion"`
	} `json:"chargingStation"`
}

func handleBootNotification(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req bootNotificationReq
	_ = json.Unmarshal(call.Payload, &req)

	status := "Accepted"
	device, err := repository.GetDevice(dc.DeviceID)
	if err != nil || !device.Enabled {
		status = "Rejected"
	}

	interval := dc.HeartbeatInterval
	if device != nil && device.HeartbeatInterval > 0 {
		interval = device.HeartbeatInterval
	}

	resp := map[string]interface{}{
		"status":      status,
		"currentTime": time.Now().UTC().Format(time.RFC3339),
		"interval":    interval,
	}
	sendResult(dc, call.MsgID, resp)

	if status == "Accepted" {
		_ = repository.UpdateDeviceStatus(dc.DeviceName, "Available")
		pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "Device boot notification accepted (2.0.1)")
	}
}

// ─── Heartbeat ───────────────────────────────────────────────────────────────
// Schema: HeartbeatRequest.json / HeartbeatResponse.json
// Response requires currentTime.

func handleHeartbeat(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	resp := map[string]interface{}{
		"currentTime": time.Now().UTC().Format(time.RFC3339),
	}
	sendResult(dc, call.MsgID, resp)
	_ = repository.UpdateDeviceHeartbeat(dc.DeviceName)
}

// ─── StatusNotification ─────────────────────────────────────────────────────
// Schema: StatusNotificationRequest.json / StatusNotificationResponse.json (empty)
// Request requires timestamp, connectorStatus, evseId, connectorId.

type statusNotificationReq struct {
	Timestamp       string `json:"timestamp"`
	ConnectorStatus string `json:"connectorStatus"`
	EvseID          int    `json:"evseId"`
	ConnectorID     int    `json:"connectorId"`
}

func handleStatusNotification(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req statusNotificationReq
	_ = json.Unmarshal(call.Payload, &req)

	_ = repository.UpdateDeviceStatus(dc.DeviceName, req.ConnectorStatus)

	sendResult(dc, call.MsgID, struct{}{})

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"StatusNotification: "+req.ConnectorStatus+" (evse="+itoa(req.EvseID)+", connector="+itoa(req.ConnectorID)+")")
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func itoa(i int) string { return strconv.Itoa(i) }

func pushEvent(eventCh chan<- *model.Event, tenantID, level, device, message string) {
	ev := &model.Event{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		Device:  device,
		Message: message,
	}
	repository.SaveEvent(ev, tenantID)
	select {
	case eventCh <- ev:
	default:
	}
}
