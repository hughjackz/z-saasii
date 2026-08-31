package repository

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
)

func ListIDTags(callerRole model.Role, callerID string, tenantID string) ([]*model.IDTag, error) {
	q := `SELECT t.*, u.name AS owner_name
		  FROM idtag t
		  LEFT JOIN role u ON u.id = t.owner_id
		  WHERE 1=1`
	args := []interface{}{}
	switch callerRole {
	case model.RoleCSAdmin:
		if tenantID != "" {
			q += " AND t.tenant_id=?"
			args = append(args, tenantID)
		}
	case model.RoleCPOP:
		q += " AND t.tenant_id=?"
		args = append(args, tenantID)
	case model.RoleCPOM:
		q += " AND t.owner_id=?"
		args = append(args, callerID)
	}
	var tags []*model.IDTag
	if err := DB.Select(&tags, q, args...); err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []*model.IDTag{}
	}
	return tags, nil
}

func GetIDTagByTagID(tagID string) (*model.IDTag, error) {
	var t model.IDTag
	err := DB.Get(&t, "SELECT * FROM idtag WHERE tag_id=? LIMIT 1", tagID)
	return &t, err
}

func CreateIDTag(t *model.IDTag) error {
	t.ID = uuid.New().String()
	_, err := DB.NamedExec(`INSERT INTO idtag (id,tag_id,parent_tag_id,status,expiry_time,owner_id,tenant_id,created_at,updated_at)
		VALUES (:id,:tag_id,:parent_tag_id,:status,:expiry_time,:owner_id,:tenant_id,NOW(),NOW())`, t)
	return err
}

func UpdateIDTag(id string, fields map[string]interface{}) error {
	q := "UPDATE idtag SET updated_at=NOW()"
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

func DeleteIDTag(id string) error {
	_, err := DB.Exec("DELETE FROM idtag WHERE id=?", id)
	return err
}
