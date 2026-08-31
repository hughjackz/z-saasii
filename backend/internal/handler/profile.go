package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

func ListProfiles(c *gin.Context) {
	callerRole, callerID, tenantID := tenantDB(c)

	q := `SELECT p.id,p.name,p.purpose,p.content,p.owner_id,p.tenant_id,p.imported_at,
		  u.name AS owner_name
		  FROM charging_profile p
		  LEFT JOIN role u ON u.id = p.owner_id
		  WHERE 1=1`
	args := []interface{}{}
	switch callerRole {
	case model.RoleCSAdmin:
		if tenantID != "" {
			q += " AND p.tenant_id=?"
			args = append(args, tenantID)
		}
	case model.RoleCPOP:
		q += " AND p.tenant_id=?"
		args = append(args, tenantID)
	case model.RoleCPOM:
		q += " AND p.owner_id=?"
		args = append(args, callerID)
	}
	var profiles []*model.ChargingProfile
	err := repository.DB.Select(&profiles, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if profiles == nil {
		profiles = []*model.ChargingProfile{}
	}
	c.JSON(http.StatusOK, profiles)
}

func UploadProfile(c *gin.Context) {
	_, _, tenantID := tenantDB(c)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, _ := file.Open()
	defer f.Close()
	content, _ := io.ReadAll(f)

	name := c.PostForm("name")
	if name == "" {
		name = file.Filename
	}

	// CS_Admin can pass tenant_id via form field; all roles: owner = tenant (CP_OP)
	tid := tenantID
	if fv := c.PostForm("tenant_id"); fv != "" { tid = fv }
	p := &model.ChargingProfile{
		ID:       uuid.New().String(),
		Name:     name,
		Purpose:  c.PostForm("purpose"),
		Content:  string(content),
		OwnerID:  tid,
		TenantID: tid,
	}
	_, err = repository.DB.NamedExec(
		`INSERT INTO charging_profile (id,name,purpose,content,owner_id,tenant_id,imported_at)
		 VALUES (:id,:name,:purpose,:content,:owner_id,:tenant_id,NOW())`, p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func RenameProfile(c *gin.Context) {
	var req struct{ Name string `json:"name"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	_, err := repository.DB.Exec("UPDATE charging_profile SET name=? WHERE id=?", req.Name, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "renamed"})
}

func DeleteProfile(c *gin.Context) {
	_, err := repository.DB.Exec("DELETE FROM charging_profile WHERE id=?", c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
