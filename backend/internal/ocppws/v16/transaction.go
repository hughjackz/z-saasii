package v16

// transaction.go — Transaction-related message handlers.
// Handles StartTransaction, StopTransaction, and MeterValues from charge points.

import (
	"encoding/json"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── StartTransaction ────────────────────────────────────────────────────────
// Schema: StartTransaction.json / StartTransactionResponse.json

type startTransactionReq struct {
	ConnectorID   int    `json:"connectorId"`
	IDTag         string `json:"idTag"`
	MeterStart    int    `json:"meterStart"`
	ReservationID *int   `json:"reservationId"`
	Timestamp     string `json:"timestamp"`
}

func handleStartTransaction(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req startTransactionReq
	_ = json.Unmarshal(call.Payload, &req)

	idTagStatus := "Accepted"
	tag, err := repository.GetIDTagByTagID(req.IDTag)
	if err == nil {
		switch tag.Status {
		case "Blocked":
			idTagStatus = "Blocked"
		case "Expired":
			idTagStatus = "Expired"
		}
	}

	txID, err := repository.GetNextTransactionID(dc.TenantID)
	if err != nil {
		idTagStatus = "Invalid"
		txID = 0
	}

	startTime := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
		startTime = t
	}

	if idTagStatus == "Accepted" {
		tx := &model.Transaction{
			TransactionID: txID,
			ChargePointID: dc.DeviceName,
			ConnectorID:   req.ConnectorID,
			TenantID:      dc.TenantID,
			IDTag:         req.IDTag,
			StartTime:     startTime,
			StartMeter:    float64(req.MeterStart),
			Active:        true,
		}
		_ = repository.UpsertTransaction(tx)
	}

	resp := map[string]interface{}{
		"idTagInfo":    map[string]interface{}{"status": idTagStatus},
		"transactionId": txID,
	}
	sendResult(dc, call.MsgID, resp)

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "StartTransaction txId="+itoa(txID)+" idTag="+req.IDTag)
}

// ─── StopTransaction ─────────────────────────────────────────────────────────
// Schema: StopTransaction.json / StopTransactionResponse.json

type stopTransactionReq struct {
	IDTag           string          `json:"idTag"`
	MeterStop       int             `json:"meterStop"`
	Timestamp       string          `json:"timestamp"`
	TransactionID   int             `json:"transactionId"`
	Reason          string          `json:"reason"`
	TransactionData json.RawMessage `json:"transactionData"`
}

func handleStopTransaction(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req stopTransactionReq
	_ = json.Unmarshal(call.Payload, &req)

	stopTime := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
		stopTime = t
	}

	_, _ = repository.DB.Exec(
		`UPDATE transaction SET stop_time=?, stop_meter=?, stop_reason=?, active=0, updated_at=NOW()
		 WHERE transaction_id=? AND charge_point_id=? AND tenant_id=? AND active=1`,
		stopTime, req.MeterStop, req.Reason, req.TransactionID, dc.DeviceName, dc.TenantID)

	resp := map[string]interface{}{
		"idTagInfo": map[string]interface{}{"status": "Accepted"},
	}
	sendResult(dc, call.MsgID, resp)

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "StopTransaction txId="+itoa(req.TransactionID)+" reason="+req.Reason)
}

// ─── MeterValues ─────────────────────────────────────────────────────────────
// Schema: MeterValues.json / MeterValuesResponse.json (empty)

type meterValuesReq struct {
	ConnectorID   int             `json:"connectorId"`
	TransactionID *int            `json:"transactionId"`
	MeterValue    json.RawMessage `json:"meterValue"`
}

func handleMeterValues(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req meterValuesReq
	_ = json.Unmarshal(call.Payload, &req)

	txID := 0
	if req.TransactionID != nil {
		txID = *req.TransactionID
	}
	_, _ = repository.DB.Exec(
		`INSERT INTO meter_value (tenant_id, transaction_id, connector_id, value, created_at)
		 VALUES (?, ?, ?, ?, NOW())`,
		dc.TenantID, txID, req.ConnectorID, string(req.MeterValue))

	sendResult(dc, call.MsgID, struct{}{})
}
