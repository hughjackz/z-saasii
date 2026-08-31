package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/config"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
	"github.com/yourorg/csms-backend/internal/vdv261"
)

// ─── VDVProfile ──────────────────────────────────────────────────────────────

// GET/POST /api/vdv/profiles
func ListVDVProfiles(c *gin.Context) {
	callerRole, _, tenantID := tenantDB(c)
	// CS_Admin: no filter → show all; with ?tenant_id= → filter by tenant
	if callerRole != model.RoleCSAdmin && tenantID == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	list, err := repository.ListVDVProfiles(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateVDVProfile(c *gin.Context) {
	callerRole, _, tenantID := tenantDB(c)
	var p model.VDVProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// CS_Admin MUST provide tenantId in request body
	if p.TenantID == "" {
		if callerRole != model.RoleCSAdmin {
			p.TenantID = tenantID
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId required"})
			return
		}
	}
	if err := repository.CreateVDVProfile(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func UpdateVDVProfile(c *gin.Context) {
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(fields, "id"); delete(fields, "tenantId"); delete(fields, "tenant_id")
	delete(fields, "createdAt"); delete(fields, "updatedAt")
	if err := repository.UpdateVDVProfile(c.Param("id"), fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteVDVProfile(c *gin.Context) {
	if err := repository.DeleteVDVProfile(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ─── VDVCarInfo ──────────────────────────────────────────────────────────────

func ListVDVCarInfos(c *gin.Context) {
	callerRole, _, tenantID := tenantDB(c)
	if callerRole != model.RoleCSAdmin && tenantID == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	list, err := repository.ListVDVCarInfos(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateVDVCarInfo(c *gin.Context) {
	callerRole, _, tenantID := tenantDB(c)
	var c2 model.VDVCarInfo
	if err := c.ShouldBindJSON(&c2); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Check duplicate VIN
	if existing, _ := repository.GetVDVCarInfoByVIN(c2.VIN); existing != nil && existing.ID != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "VIN already exists: " + c2.VIN})
		return
	}
	if c2.TenantID == "" {
		if callerRole != model.RoleCSAdmin {
			c2.TenantID = tenantID
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId required"})
			return
		}
	}
	if err := repository.CreateVDVCarInfo(&c2); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, c2)
}

func UpdateVDVCarInfo(c *gin.Context) {
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(fields, "id"); delete(fields, "vin"); delete(fields, "tenantId"); delete(fields, "tenant_id")
	delete(fields, "createdAt"); delete(fields, "updatedAt")
	// Map vdvProfileId → vdv_profile_id
	if v, ok := fields["vdvProfileId"]; ok {
		fields["vdv_profile_id"] = v
		delete(fields, "vdvProfileId")
	}
	if err := repository.UpdateVDVCarInfo(c.Param("id"), fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteVDVCarInfo(c *gin.Context) {
	if err := repository.DeleteVDVCarInfo(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ─── VDV261 Settings (6.2.2 / 6.2.3, CS_Admin only) ──────────────────────

func GetVDVSettings(c *gin.Context) {
	c.JSON(http.StatusOK, config.Cfg.VDV261)
}

func UpdateVDVSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Apply settings to runtime config
	if v, ok := req["network_mode"].(string); ok {
		config.Cfg.VDV261.NetworkMode = v
	}
	if v, ok := req["enable"].(bool); ok {
		config.Cfg.VDV261.Enable = v
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

func RestartVDVService(c *gin.Context) {
	go func() {
		vdv261.Start()
	}()
	c.JSON(http.StatusOK, gin.H{"message": "VDV261 service restart initiated"})
}

// POST /api/vdv/settings/upload-cert — upload VDV cert/key files
func UploadVDVCert(c *gin.Context) {
	certType := c.PostForm("type") // vdv-root, vdv-server-cert, vdv-server-key
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, _ := file.Open()
	defer f.Close()
	content, _ := io.ReadAll(f)

	baseDir := "resource/admin/cert"
	_ = os.MkdirAll(baseDir, 0755)

	var filename string
	switch certType {
	case "vdv-root":
		filename = "VDVroot.pem"
	case "vdv-server-cert":
		filename = "VDVserver.pem"
	case "vdv-server-key":
		filename = "VDVserver.key"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}

	_ = os.WriteFile(filepath.Join(baseDir, filename), content, 0644)
	c.JSON(http.StatusOK, gin.H{"message": filename + " uploaded"})
}

// GET /api/vdv/settings/download-cert?type=vdv-root — download cert/key
func DownloadVDVCert(c *gin.Context) {
	certType := c.Query("type")
	var filename string
	switch certType {
	case "vdv-root": filename = "VDVroot.pem"
	case "vdv-server-cert": filename = "VDVserver.pem"
	case "vdv-server-key": filename = "VDVserver.key"
	default: c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"}); return
	}
	c.File(filepath.Join("resource/admin/cert", filename))
}
