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
func ListTransactions(callerRole model.Role, callerID string, tenantID string, deviceID string, page, limit int) (*TransactionPage, error) {
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
