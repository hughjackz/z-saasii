package v16

// conn.go — Device connection management handlers.
// Handles BootNotification and Heartbeat from charge points.

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── BootNotification ────────────────────────────────────────────────────────
// Schema: BootNotification.json / BootNotificationResponse.json

type bootNotificationReq struct {
	ChargePointVendor       string `json:"chargePointVendor"`
	ChargePointModel        string `json:"chargePointModel"`
	ChargePointSerialNumber string `json:"chargePointSerialNumber"`
	ChargeBoxSerialNumber   string `json:"chargeBoxSerialNumber"`
	FirmwareVersion         string `json:"firmwareVersion"`
	Iccid                   string `json:"iccid"`
	Imsi                    string `json:"imsi"`
	MeterType               string `json:"meterType"`
	MeterSerialNumber       string `json:"meterSerialNumber"`
}

func handleBootNotification(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req bootNotificationReq
	_ = json.Unmarshal(call.Payload, &req)

	status := "Accepted"
	device, err := repository.GetDevice(dc.DeviceID)
	if err != nil || !device.Enabled {
		status = "Rejected"
	}

	resp := map[string]interface{}{
		"status":      status,
		"currentTime": time.Now().UTC().Format(time.RFC3339),
		"interval":    device.HeartbeatInterval,
	}

	sendResult(dc, call.MsgID, resp)

	if status == "Accepted" {
		_ = repository.UpdateDeviceStatus(device.Name, "Available")
		pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "Device boot notification accepted")
	}
}

// ─── Heartbeat ───────────────────────────────────────────────────────────────
// Schema: Heartbeat.json / HeartbeatResponse.json

func handleHeartbeat(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	resp := map[string]interface{}{
		"currentTime": time.Now().UTC().Format(time.RFC3339),
	}
	sendResult(dc, call.MsgID, resp)
	_ = repository.UpdateDeviceHeartbeat(dc.DeviceName)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func init() {
	ocppws.V16Handler = HandleCall
}
