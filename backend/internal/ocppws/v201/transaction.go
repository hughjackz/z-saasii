package v201

// transaction.go — TransactionEvent handler (README 4.3.5).
// OCPP 2.0.1 replaces StartTransaction/StopTransaction/MeterValues with a
// single TransactionEvent message carrying eventType Started/Updated/Ended.

import (
	"encoding/json"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── TransactionEvent ────────────────────────────────────────────────────────
// Schema: TransactionEventRequest.json / TransactionEventResponse.json

type transactionEventReq struct {
	EventType       string `json:"eventType"`
	Timestamp       string `json:"timestamp"`
	TriggerReason   string `json:"triggerReason"`
	SeqNo           int    `json:"seqNo"`
	TransactionInfo struct {
		TransactionID string `json:"transactionId"`
		ChargingState string `json:"chargingState"`
		StoppedReason string `json:"stoppedReason"`
	} `json:"transactionInfo"`
	Evse *struct {
		ID          int  `json:"id"`
		ConnectorID *int `json:"connectorId"`
	} `json:"evse"`
	IDToken *struct {
		IDToken string `json:"idToken"`
		Type    string `json:"type"`
	} `json:"idToken"`
	MeterValue []meterValueType `json:"meterValue"`
}

type meterValueType struct {
	Timestamp    string             `json:"timestamp"`
	SampledValue []sampledValueType `json:"sampledValue"`
}

type sampledValueType struct {
	Value     float64 `json:"value"`
	Measurand string  `json:"measurand"`
}

func handleTransactionEvent(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req transactionEventReq
	_ = json.Unmarshal(call.Payload, &req)

	connectorID := 1
	if req.Evse != nil && req.Evse.ConnectorID != nil {
		connectorID = *req.Evse.ConnectorID
	}
	idTag := ""
	if req.IDToken != nil {
		idTag = req.IDToken.IDToken
	}

	eventTime := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
		eventTime = t
	}

	switch req.EventType {
	case "Started":
		handleTxStarted(dc, call, eventCh, &req, connectorID, idTag, eventTime)
	case "Ended":
		handleTxEnded(dc, call, eventCh, &req, connectorID, eventTime)
	case "Updated":
		handleTxUpdated(dc, call, eventCh, &req, connectorID)
	default:
		// Unknown eventType — acknowledge without side effects
		sendResult(dc, call.MsgID, struct{}{})
	}
}

// handleTxStarted generates the platform's own transaction id (int, auto-
// increment per tenant) and persists the transaction row (README 4.2.7.3).
func handleTxStarted(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event,
	req *transactionEventReq, connectorID int, idTag string, eventTime time.Time) {

	txID, err := repository.GetNextTransactionID(dc.TenantID)
	if err != nil {
		txID = 0
	}

	tx := &model.Transaction{
		TransactionID: txID,
		ChargePointID: dc.DeviceName,
		ConnectorID:   connectorID,
		TenantID:      dc.TenantID,
		IDTag:         idTag,
		StartTime:     eventTime,
		StartMeter:    firstMeterValue(req),
		Active:        true,
	}
	_ = repository.UpsertTransaction(tx)

	sendResult(dc, call.MsgID, struct{}{})

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"TransactionEvent Started txId="+itoa(txID)+" idTag="+idTag)
}

// handleTxUpdated persists meter values sampled during the transaction.
func handleTxUpdated(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event,
	req *transactionEventReq, connectorID int) {

	if len(req.MeterValue) > 0 {
		valueJSON, _ := json.Marshal(req.MeterValue)
		_, _ = repository.DB.Exec(
			`INSERT INTO meter_value (tenant_id, transaction_id, connector_id, value, created_at)
			 VALUES (?, ?, ?, ?, NOW())`,
			dc.TenantID, 0, connectorID, string(valueJSON))
	}

	sendResult(dc, call.MsgID, struct{}{})
}

// handleTxEnded closes the active transaction for this charge point.
// The device reports its own (string) transactionId; the platform matches by
// charge point + tenant and stops the latest active row.
func handleTxEnded(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event,
	req *transactionEventReq, connectorID int, eventTime time.Time) {

	_, _ = repository.DB.Exec(
		`UPDATE transaction SET stop_time=?, stop_meter=?, stop_reason=?, active=0, updated_at=NOW()
		 WHERE charge_point_id=? AND tenant_id=? AND active=1`,
		eventTime, lastMeterValue(req), req.TransactionInfo.StoppedReason, dc.DeviceName, dc.TenantID)

	sendResult(dc, call.MsgID, struct{}{})

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"TransactionEvent Ended deviceTxId="+req.TransactionInfo.TransactionID+" reason="+req.TransactionInfo.StoppedReason)
}

// firstMeterValue returns the first sampled value with measurand
// Energy.Active.Import.Register (fallback: first sampled value overall).
func firstMeterValue(req *transactionEventReq) float64 {
	for _, mv := range req.MeterValue {
		for _, sv := range mv.SampledValue {
			if sv.Measurand == "Energy.Active.Import.Register" || sv.Measurand == "" {
				return sv.Value
			}
		}
	}
	return lastMeterValue(req)
}

// lastMeterValue returns the last sampled value reported in this event.
func lastMeterValue(req *transactionEventReq) float64 {
	var v float64
	for _, mv := range req.MeterValue {
		for _, sv := range mv.SampledValue {
			v = sv.Value
		}
	}
	return v
}
