package repository

import (
	"github.com/yourorg/csms-backend/internal/model"
)

// ListActiveTransactions returns active transactions filtered by role and tenant:
//   - CS_Admin: no direct transaction access (returns empty)
//   - CP_OP: all active tx in their tenant
//   - CP_OM: active tx on their own devices (owner_id = callerID)
func ListActiveTransactions(callerRole model.Role, callerID string, tenantID string, deviceIDFilter string) ([]*model.Transaction, error) {
	q := "SELECT * FROM transaction WHERE active=1"
	args := []interface{}{}
	if deviceIDFilter != "" {
		q += " AND charge_point_id=?"
		args = append(args, deviceIDFilter)
	}
	switch callerRole {
	case model.RoleCSAdmin:
		return []*model.Transaction{}, nil
	case model.RoleCPOP:
		q += " AND tenant_id=?"
		args = append(args, callerID)
	case model.RoleCPOM:
		q += " AND charge_point_id IN (SELECT name FROM device WHERE owner_id=?)"
		args = append(args, callerID)
	}
	var txs []*model.Transaction
	if err := DB.Select(&txs, q, args...); err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []*model.Transaction{}
	}
	return txs, nil
}

type TransactionPage struct {
	Data       []*model.Transaction `json:"data"`
	Total      int                  `json:"total"`
	TotalPages int                  `json:"totalPages"`
}

// ListTransactions returns historical (inactive) transactions filtered by role and tenant.
// transactionIDFilter optionally filters by transaction_id (README 2.3.2.2.1).
func ListTransactions(callerRole model.Role, callerID string, tenantID string, deviceID string, transactionIDFilter string, page, limit int) (*TransactionPage, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	base := "FROM transaction WHERE active=0"
	args := []interface{}{}
	if deviceID != "" {
		base += " AND charge_point_id=?"
		args = append(args, deviceID)
	}
	if transactionIDFilter != "" {
		base += " AND transaction_id=?"
		args = append(args, transactionIDFilter)
	}
	switch callerRole {
	case model.RoleCSAdmin:
		return &TransactionPage{Data: []*model.Transaction{}, Total: 0, TotalPages: 0}, nil
	case model.RoleCPOP:
		base += " AND tenant_id=?"
		args = append(args, callerID)
	case model.RoleCPOM:
		base += " AND charge_point_id IN (SELECT name FROM device WHERE owner_id=?)"
		args = append(args, callerID)
	}

	var total int
	_ = DB.Get(&total, "SELECT COUNT(*) "+base, args...)

	totalPages := (total + limit - 1) / limit

	qArgs := append(args, limit, offset)
	var txs []*model.Transaction
	err := DB.Select(&txs, "SELECT * "+base+" ORDER BY start_time DESC LIMIT ? OFFSET ?", qArgs...)
	if txs == nil {
		txs = []*model.Transaction{}
	}
	return &TransactionPage{Data: txs, Total: total, TotalPages: totalPages}, err
}

// GetNextTransactionID returns the next transaction_id for a given tenant.
func GetNextTransactionID(tenantID string) (int, error) {
	var maxID int
	err := DB.Get(&maxID, "SELECT COALESCE(MAX(transaction_id), 0) + 1 FROM transaction WHERE tenant_id=?", tenantID)
	return maxID, err
}

func UpsertTransaction(tx *model.Transaction) error {
	_, err := DB.NamedExec(`INSERT INTO transaction
		(transaction_id,charge_point_id,connector_id,tenant_id,id_tag,start_time,start_meter,active)
		VALUES (:transaction_id,:charge_point_id,:connector_id,:tenant_id,:id_tag,:start_time,:start_meter,1)
		ON DUPLICATE KEY UPDATE stop_time=VALUES(stop_time), stop_meter=VALUES(stop_meter),
		stop_reason=VALUES(stop_reason), active=VALUES(active)`, tx)
	return err
}

// ─── OCPP 2.0.1 transactions (table transaction_201) ───────────────────────
// transactionId is device-generated (string); a transaction is bound by
// charge_point_id + evse_id + connector_id + transaction_id (README 4.3.5).

// Tx201Filter carries optional filters for OCPP 2.0.1 transaction queries.
type Tx201Filter struct {
	TransactionID string // filter by device-generated transactionId
	ShowMeterOnly bool   // internal: include/exclude nothing, kept for clarity
}

func ListActiveTransactions201(callerRole model.Role, callerID string, tenantID string, deviceIDFilter string) ([]*model.Transaction201, error) {
	q := "SELECT * FROM transaction_201 WHERE active=1"
	args := []interface{}{}
	if deviceIDFilter != "" {
		q += " AND charge_point_id=?"
		args = append(args, deviceIDFilter)
	}
	switch callerRole {
	case model.RoleCSAdmin:
		return []*model.Transaction201{}, nil
	case model.RoleCPOP:
		q += " AND tenant_id=?"
		args = append(args, callerID)
	case model.RoleCPOM:
		q += " AND charge_point_id IN (SELECT name FROM device WHERE owner_id=?)"
		args = append(args, callerID)
	}
	var txs []*model.Transaction201
	if err := DB.Select(&txs, q, args...); err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []*model.Transaction201{}
	}
	return txs, nil
}

// Transaction201Page is the paged response for OCPP 2.0.1 transactions.
type Transaction201Page struct {
	Data       []*model.Transaction201 `json:"data"`
	Total      int                     `json:"total"`
	TotalPages int                     `json:"totalPages"`
}

// ListTransactions201 returns historical (inactive) OCPP 2.0.1 transactions.
// Optional filters: transactionID (README 2.3.2.2.1 transactionid 筛选器).
func ListTransactions201(callerRole model.Role, callerID string, tenantID string, deviceID string, transactionIDFilter string, page, limit int) (*Transaction201Page, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	base := "FROM transaction_201 WHERE active=0"
	args := []interface{}{}
	if deviceID != "" {
		base += " AND charge_point_id=?"
		args = append(args, deviceID)
	}
	if transactionIDFilter != "" {
		base += " AND transaction_id=?"
		args = append(args, transactionIDFilter)
	}
	switch callerRole {
	case model.RoleCSAdmin:
		return &Transaction201Page{Data: []*model.Transaction201{}, Total: 0, TotalPages: 0}, nil
	case model.RoleCPOP:
		base += " AND tenant_id=?"
		args = append(args, callerID)
	case model.RoleCPOM:
		base += " AND charge_point_id IN (SELECT name FROM device WHERE owner_id=?)"
		args = append(args, callerID)
	}

	var total int
	_ = DB.Get(&total, "SELECT COUNT(*) "+base, args...)
	totalPages := (total + limit - 1) / limit

	qArgs := append(args, limit, offset)
	var txs []*model.Transaction201
	err := DB.Select(&txs, "SELECT * "+base+" ORDER BY start_time DESC LIMIT ? OFFSET ?", qArgs...)
	if txs == nil {
		txs = []*model.Transaction201{}
	}
	return &Transaction201Page{Data: txs, Total: total, TotalPages: totalPages}, err
}

// UpsertTransaction201 inserts or updates an OCPP 2.0.1 transaction row.
func UpsertTransaction201(tx *model.Transaction201) error {
	_, err := DB.NamedExec(`INSERT INTO transaction_201
		(transaction_id,charge_point_id,evse_id,connector_id,tenant_id,id_tag,start_time,start_meter,active)
		VALUES (:transaction_id,:charge_point_id,:evse_id,:connector_id,:tenant_id,:id_tag,:start_time,:start_meter,1)
		ON DUPLICATE KEY UPDATE id_tag=VALUES(id_tag), start_time=VALUES(start_time),
			start_meter=VALUES(start_meter), active=1`, tx)
	return err
}

// CloseTransaction201 finishes the transaction bound by
// charge_point_id+evse_id+connector_id+transaction_id.
func CloseTransaction201(chargePointID string, evseID, connectorID int, transactionID string, tenantID string, stopTime interface{}, stopMeter float64, stopReason string) error {
	_, err := DB.Exec(`UPDATE transaction_201
		SET stop_time=?, stop_meter=?, stop_reason=?, active=0
		WHERE charge_point_id=? AND evse_id=? AND connector_id=? AND transaction_id=? AND tenant_id=?`,
		stopTime, stopMeter, stopReason, chargePointID, evseID, connectorID, transactionID, tenantID)
	return err
}

// InsertTransactionEvent201 records one raw TransactionEvent message (4.3.5).
func InsertTransactionEvent201(ev *model.TransactionEvent201) error {
	_, err := DB.Exec(`INSERT INTO transaction_event_201
		(tenant_id,charge_point_id,transaction_id,evse_id,connector_id,event_type,trigger_reason,seq_no,payload,created_at)
		VALUES (?,?,?,?,?,?,?,?,?,NOW())`,
		ev.TenantID, ev.ChargePointID, ev.TransactionID, ev.EvseID, ev.ConnectorID,
		ev.EventType, ev.TriggerReason, ev.SeqNo, ev.Payload)
	return err
}

// ListTransactionEvents201 returns recorded TransactionEvents for a device.
// Meter events (triggerReason MeterValueClock/MeterValuePeriodic) are excluded
// when includeMeter is false — they only go to the event log (README 4.3.5).
func ListTransactionEvents201(deviceName string, tenantID string, includeMeter bool, limit int) ([]*model.TransactionEvent201, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT * FROM transaction_event_201 WHERE charge_point_id=?`
	args := []interface{}{deviceName}
	if tenantID != "" {
		q += " AND tenant_id=?"
		args = append(args, tenantID)
	}
	if !includeMeter {
		q += " AND trigger_reason NOT IN ('MeterValueClock','MeterValuePeriodic')"
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)
	var rows []*model.TransactionEvent201
	if err := DB.Select(&rows, q, args...); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*model.TransactionEvent201{}
	}
	return rows, nil
}
