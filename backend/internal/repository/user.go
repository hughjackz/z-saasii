package repository

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	err := DB.Get(&u, "SELECT * FROM role WHERE username=? LIMIT 1", username)
	return &u, err
}

func GetUserByID(id string) (*model.User, error) {
	var u model.User
	err := DB.Get(&u, "SELECT * FROM role WHERE id=? LIMIT 1", id)
	return &u, err
}

// ListUsers returns users based on caller's role:
//   - CS_Admin: only sees CP_OP users
//   - CP_OP: only sees CP_OM users whose parent_id = callerID
//   - CP_OM: only sees themselves
func ListUsers(callerRole model.Role, callerID string, roleFilter string) ([]*model.User, error) {
	q := "SELECT * FROM role WHERE 1=1"
	args := []interface{}{}
	switch callerRole {
	case model.RoleCSAdmin:
		// CS_Admin sees CP_OP and CP_OM users (not self)
		if roleFilter != "" {
			q += " AND role=?"
			args = append(args, roleFilter)
		} else {
			q += " AND role IN (?, ?)"
			args = append(args, model.RoleCPOP, model.RoleCPOM)
		}
		// Return early — don't add the extra roleFilter below
		var users []*model.User
		if err := DB.Select(&users, q, args...); err != nil {
			return nil, err
		}
		if users == nil {
			users = []*model.User{}
		}
		return users, nil
	case model.RoleCPOP:
		// CP_OP only sees CP_OM users under them
		q += " AND parent_id=? AND role=?"
		args = append(args, callerID, model.RoleCPOM)
	case model.RoleCPOM:
		// CP_OM only sees themselves
		q += " AND id=?"
		args = append(args, callerID)
	}
	if roleFilter != "" && callerRole != model.RoleCSAdmin && callerRole != model.RoleCPOP {
		// Only apply extra role filter for non-scoped queries
		q += " AND role=?"
		args = append(args, roleFilter)
	}
	var users []*model.User
	if err := DB.Select(&users, q, args...); err != nil {
		return nil, err
	}
	if users == nil {
		users = []*model.User{}
	}
	return users, nil
}

// ListCPOPs returns all CP_OP users (for CS_Admin dropdowns etc.)
func ListCPOPs() ([]*model.User, error) {
	var users []*model.User
	err := DB.Select(&users, "SELECT * FROM role WHERE role=? ORDER BY username", model.RoleCPOP)
	if users == nil {
		users = []*model.User{}
	}
	return users, err
}

// ListCPOMsByParent returns all CP_OM users under a given CP_OP
func ListCPOMsByParent(cpopID string) ([]*model.User, error) {
	var users []*model.User
	err := DB.Select(&users, "SELECT * FROM role WHERE role=? AND parent_id=? ORDER BY username",
		model.RoleCPOM, cpopID)
	if users == nil {
		users = []*model.User{}
	}
	return users, err
}

// CreateUser creates a new user with proper tenant_id calculation:
//   - CS_Admin creating CP_OP: tenant_id = new user's own id
//   - CP_OP creating CP_OM: tenant_id = CP_OP's id (i.e. caller's id)
func CreateUser(req *model.UserResponse, plainPwd string, callerID string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	permsJSON, _ := json.Marshal(req.Permissions)

	newID := uuid.New().String()

	// parent_id: only CP_OM has a parent (the CP_OP)
	var parentID *string
	if req.Role == model.RoleCPOM && req.ParentID != nil && *req.ParentID != "" {
		parentID = req.ParentID
	}

	// tenant_id logic:
	//   CP_OP: tenant_id = own id (they are the tenant root)
	//   CP_OM: tenant_id = parent CP_OP's id (passed via req.TenantID or calculated)
	//   CS_Admin: tenant_id = NULL
	var tenantID *string
	switch req.Role {
	case model.RoleCPOP:
		tenantID = &newID
	case model.RoleCPOM:
		if req.TenantID != nil && *req.TenantID != "" {
			tenantID = req.TenantID
		} else if req.ParentID != nil && *req.ParentID != "" {
			tenantID = req.ParentID // parent CP_OP's id IS the tenant_id
		}
	}

	createdBy := &callerID

	u := &model.User{
		ID:           newID,
		Username:     req.Username,
		PasswordHash: string(hash),
		Name:         req.Name,
		Role:         req.Role,
		Company:      req.Company,
		Email:        req.Email,
		Contact:      req.Contact,
		Enabled:      req.Enabled,
		ParentID:     parentID,
		TenantID:     tenantID,
		CreatedBy:    createdBy,
		Permissions:  string(permsJSON),
	}

	_, err = DB.NamedExec(`
		INSERT INTO role
		  (id, username, password_hash, name, role, company, email, contact,
		   enabled, parent_id, tenant_id, created_by, permissions, created_at, updated_at)
		VALUES
		  (:id, :username, :password_hash, :name, :role, :company, :email, :contact,
		   :enabled, :parent_id, :tenant_id, :created_by, :permissions, NOW(), NOW())`, u)
	return u, err
}

func UpdateUser(id string, req map[string]interface{}) error {
	if perms, ok := req["permissions"]; ok {
		b, _ := json.Marshal(perms)
		req["permissions"] = string(b)
	}

	// parentId (JSON) → parent_id (DB column); empty string → NULL
	if v, ok := req["parentId"]; ok {
		s, isStr := v.(string)
		if !isStr || s == "" {
			req["parent_id"] = nil
		} else {
			req["parent_id"] = s
		}
		delete(req, "parentId")
	}

	// tenantId (JSON) → tenant_id (DB column)
	if v, ok := req["tenantId"]; ok {
		s, isStr := v.(string)
		if !isStr || s == "" {
			req["tenant_id"] = nil
		} else {
			req["tenant_id"] = s
		}
		delete(req, "tenantId")
	}

	if pwd, ok := req["password"].(string); ok && pwd != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		req["password_hash"] = string(hash)
		delete(req, "password")
	}

	// 永远不允许通过此接口修改 created_by
	delete(req, "createdBy")
	delete(req, "created_by")

	q := "UPDATE role SET updated_at=NOW()"
	args := []interface{}{}
	for k, v := range req {
		q += fmt.Sprintf(", %s=?", k)
		args = append(args, v)
	}
	q += " WHERE id=?"
	args = append(args, id)
	_, err := DB.Exec(q, args...)
	return err
}

func UpdateUserPermissions(id string, permissions []string) error {
	b, _ := json.Marshal(permissions)
	_, err := DB.Exec("UPDATE role SET permissions=?, updated_at=NOW() WHERE id=?", string(b), id)
	return err
}

func DeleteUser(id string) error {
	_, err := DB.Exec("DELETE FROM role WHERE id=?", id)
	return err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func ParsePermissions(raw string) []string {
	var perms []string
	_ = json.Unmarshal([]byte(raw), &perms)
	return perms
}

func ToUserResponse(u *model.User) model.UserResponse {
	return model.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Name:        u.Name,
		Role:        u.Role,
		Company:     u.Company,
		Email:       u.Email,
		Contact:     u.Contact,
		Enabled:     u.Enabled,
		ParentID:    u.ParentID,
		TenantID:    u.TenantID,
		CreatedBy:   u.CreatedBy,
		Permissions: ParsePermissions(u.Permissions),
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
