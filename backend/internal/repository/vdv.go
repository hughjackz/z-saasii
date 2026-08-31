package repository

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
)

// ─── VDVProfile ──────────────────────────────────────────────────────────────

func ListVDVProfiles(tenantID string) ([]*model.VDVProfile, error) {
	var list []*model.VDVProfile
	q := `SELECT v.*, r.name AS cpop_name FROM vdv_profile v LEFT JOIN role r ON r.id=v.tenant_id`
	args := []interface{}{}
	if tenantID != "" {
		q += " WHERE v.tenant_id=?"
		args = append(args, tenantID)
	}
	q += " ORDER BY v.name"
	err := DB.Select(&list, q, args...)
	if list == nil { list = []*model.VDVProfile{} }
	return list, err
}

func GetVDVProfile(id string) (*model.VDVProfile, error) {
	var p model.VDVProfile
	err := DB.Get(&p, "SELECT * FROM vdv_profile WHERE id=?", id)
	return &p, err
}

func CreateVDVProfile(p *model.VDVProfile) error {
	p.ID = uuid.New().String()
	_, err := DB.NamedExec(`INSERT INTO vdv_profile (id,name,driveoff,prec_dsrd,prec_hvac,ambienttemp,tenant_id,created_at,updated_at)
		VALUES (:id,:name,:driveoff,:prec_dsrd,:prec_hvac,:ambienttemp,:tenant_id,NOW(),NOW())`, p)
	return err
}

func UpdateVDVProfile(id string, fields map[string]interface{}) error {
	q := "UPDATE vdv_profile SET updated_at=NOW()"
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

func DeleteVDVProfile(id string) error {
	_, err := DB.Exec("DELETE FROM vdv_profile WHERE id=?", id)
	return err
}

// ─── VDVCarInfo ──────────────────────────────────────────────────────────────

func ListVDVCarInfos(tenantID string) ([]*model.VDVCarInfo, error) {
	var list []*model.VDVCarInfo
	q := `SELECT c.*, p.name AS vdv_profile_name, r.name AS cpop_name
		  FROM vdv_carinfo c
		  LEFT JOIN vdv_profile p ON p.id=c.vdv_profile_id
		  LEFT JOIN role r ON r.id=c.tenant_id`
	args := []interface{}{}
	if tenantID != "" {
		q += " WHERE c.tenant_id=?"
		args = append(args, tenantID)
	}
	q += " ORDER BY c.vin"
	err := DB.Select(&list, q, args...)
	if list == nil { list = []*model.VDVCarInfo{} }
	return list, err
}

func GetVDVCarInfoByVIN(vin string) (*model.VDVCarInfo, error) {
	var c model.VDVCarInfo
	err := DB.Get(&c, "SELECT * FROM vdv_carinfo WHERE vin=? LIMIT 1", vin)
	return &c, err
}

func CreateVDVCarInfo(c *model.VDVCarInfo) error {
	c.ID = uuid.New().String()
	_, err := DB.NamedExec(`INSERT INTO vdv_carinfo (id,vin,password,evccid,odo,vdv_profile_id,tenant_id,created_at,updated_at)
		VALUES (:id,:vin,:password,:evccid,:odo,:vdv_profile_id,:tenant_id,NOW(),NOW())`, c)
	return err
}

func UpdateVDVCarInfo(id string, fields map[string]interface{}) error {
	q := "UPDATE vdv_carinfo SET updated_at=NOW()"
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

func DeleteVDVCarInfo(id string) error {
	_, err := DB.Exec("DELETE FROM vdv_carinfo WHERE id=?", id)
	return err
}
