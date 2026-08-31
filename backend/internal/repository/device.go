package repository

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
)

func ListDevices(callerRole model.Role, callerID string, tenantID string) ([]*model.Device, error) {
	q := `SELECT d.*, u.name AS owner_name FROM device d LEFT JOIN role u ON u.id = d.owner_id WHERE 1=1`
	args := []interface{}{}
	switch callerRole {
	case model.RoleCSAdmin:
		if tenantID != "" {
			q += " AND d.tenant_id=?"
			args = append(args, tenantID)
		}
	case model.RoleCPOP:
		q += " AND d.tenant_id=?"
		args = append(args, callerID)
	case model.RoleCPOM:
		q += " AND d.owner_id=?"
		args = append(args, callerID)
	}
	var devices []*model.Device
	if err := DB.Select(&devices, q, args...); err != nil {
		return nil, err
	}
	if devices == nil {
		devices = []*model.Device{}
	}
	return devices, nil
}

func GetDevice(id string) (*model.Device, error) {
	var d model.Device
	err := DB.Get(&d, `SELECT d.*, u.name AS owner_name FROM device d LEFT JOIN role u ON u.id=d.owner_id WHERE d.id=?`, id)
	return &d, err
}

func GetDeviceByName(name string) (*model.Device, error) {
	var d model.Device
	err := DB.Get(&d, `SELECT d.*, u.name AS owner_name FROM device d LEFT JOIN role u ON u.id=d.owner_id WHERE d.name=?`, name)
	return &d, err
}

func CreateDevice(req *model.Device) (*model.Device, error) {
	req.ID = uuid.New().String()
	req.Status = "Unavailable"
	_, err := DB.NamedExec(`INSERT INTO device
		(id,name,protocol,location,enabled,heartbeat_interval,owner_id,tenant_id,status,created_at,updated_at)
		VALUES (:id,:name,:protocol,:location,:enabled,:heartbeat_interval,:owner_id,:tenant_id,:status,NOW(),NOW())`, req)
	return req, err
}

func UpdateDevice(id string, fields map[string]interface{}) error {
	q := "UPDATE device SET updated_at=NOW()"
	args := []interface{}{}
	for k, v := range fields {
		q += fmt.Sprintf(", %s=?", k)
		args = append(args, v)
	}
	q += " WHERE id=?"
	args = append(args, id)
	_, err := DB.Exec(q, args...)
	return err
}

func DeleteDevice(id string) error {
	_, err := DB.Exec("DELETE FROM device WHERE id=?", id)
	return err
}

func UpdateDeviceStatus(deviceName, status string) error {
	_, err := DB.Exec("UPDATE device SET status=?,last_heartbeat=NOW(),updated_at=NOW() WHERE name=?", status, deviceName)
	return err
}

func UpdateDeviceHeartbeat(deviceName string) error {
	_, err := DB.Exec("UPDATE device SET last_heartbeat=NOW(),updated_at=NOW() WHERE name=?", deviceName)
	return err
}
