package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/middleware"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

// GET /api/users
func ListUsers(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	roleFilter := c.Query("role")

	users, err := repository.ListUsers(callerRole, callerID, roleFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]model.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, repository.ToUserResponse(u))
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/users/cpops — returns all CP_OP users (for dropdowns)
func ListCPOPs(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	if callerRole != model.RoleCSAdmin && callerRole != model.RoleCPOP {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}
	users, err := repository.ListCPOPs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]model.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, repository.ToUserResponse(u))
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/users/cpoms — returns CP_OM users under a given CP_OP
func ListCPOMs(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	parentID := c.Query("parent")

	switch callerRole {
	case model.RoleCSAdmin:
		if parentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent query param required"})
			return
		}
		callerID = parentID
	case model.RoleCPOP:
		// return CP_OMs under self
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	users, err := repository.ListCPOMsByParent(callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]model.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, repository.ToUserResponse(u))
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/users/:id
func GetUser(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	targetID := c.Param("id")

	user, err := repository.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if callerRole == model.RoleCPOM && targetID != callerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}
	if callerRole == model.RoleCPOP && user.ParentID != nil && *user.ParentID != callerID && targetID != callerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	c.JSON(http.StatusOK, repository.ToUserResponse(user))
}

// POST /api/users
func CreateUser(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)

	var req struct {
		model.UserResponse
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch callerRole {
	case model.RoleCSAdmin:
		if req.Role == model.RoleCPOP {
			req.ParentID = nil
		} else if req.Role == model.RoleCPOM {
			if req.ParentID == nil || *req.ParentID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "创建 CP_OM 时必须选择所属 CP_OP"})
				return
			}
			req.TenantID = req.ParentID
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "CS_Admin 只能创建 CP_OP 或 CP_OM"})
			return
		}
	case model.RoleCPOP:
		if req.Role != model.RoleCPOM {
			c.JSON(http.StatusForbidden, gin.H{"error": "CP_OP 只能创建 CP_OM"})
			return
		}
		req.ParentID = &callerID
		req.TenantID = &callerID
	case model.RoleCPOM:
		c.JSON(http.StatusForbidden, gin.H{"error": "CP_OM 无权创建用户"})
		return
	}

	user, err := repository.CreateUser(&req.UserResponse, req.Password, callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, repository.ToUserResponse(user))
}

// PUT /api/users/:id
func UpdateUser(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	targetID := c.Param("id")

	targetUser, err := repository.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch callerRole {
	case model.RoleCSAdmin:
		if targetUser.Role != model.RoleCPOP && targetUser.Role != model.RoleCPOM {
			c.JSON(http.StatusForbidden, gin.H{"error": "CS_Admin 只能修改 CP_OP 或 CP_OM 用户"})
			return
		}
	case model.RoleCPOP:
		if targetUser.Role != model.RoleCPOM || targetUser.ParentID == nil || *targetUser.ParentID != callerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "CP_OP 只能修改自己的 CP_OM 用户"})
			return
		}
		delete(fields, "role")
	case model.RoleCPOM:
		if targetID == callerID {
			allowed := map[string]bool{"password": true}
			for k := range fields {
				if !allowed[k] {
					delete(fields, k)
				}
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "CP_OM 只能修改自己的密码"})
			return
		}
	}

	for _, f := range []string{
		"id", "username", "createdBy", "created_by",
		"createdAt", "created_at",
		"updatedAt", "updated_at",
		"permissions",
		"tenant_id", "tenantId",
	} {
		delete(fields, f)
	}

	if err := repository.UpdateUser(targetID, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user, _ := repository.GetUserByID(targetID)
	c.JSON(http.StatusOK, repository.ToUserResponse(user))
}

// PUT /api/users/:id/permissions
func UpdateUserPermissions(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	targetID := c.Param("id")

	targetUser, err := repository.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch callerRole {
	case model.RoleCSAdmin:
		if targetUser.Role != model.RoleCPOP && targetUser.Role != model.RoleCPOM {
			c.JSON(http.StatusForbidden, gin.H{"error": "CS_Admin 只能管理 CP_OP 或 CP_OM 的权限"})
			return
		}
	case model.RoleCPOP:
		if targetUser.Role != model.RoleCPOM || targetUser.ParentID == nil || *targetUser.ParentID != callerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "CP_OP 只能管理自己名下 CP_OM 的权限"})
			return
		}
		caller, err := repository.GetUserByID(callerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		callerPerms := repository.ParsePermissions(caller.Permissions)
		permSet := make(map[string]bool, len(callerPerms))
		for _, p := range callerPerms {
			permSet[p] = true
		}
		for _, p := range req.Permissions {
			if !permSet[p] {
				c.JSON(http.StatusForbidden, gin.H{"error": "cannot assign permission you don't have: " + p})
				return
			}
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	if err := repository.UpdateUserPermissions(targetID, req.Permissions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permissions updated"})
}

// DELETE /api/users/:id
func DeleteUser(c *gin.Context) {
	callerRole := model.Role(c.GetString(middleware.CtxRole))
	callerID := c.GetString(middleware.CtxUserID)
	targetID := c.Param("id")

	if targetID == callerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	targetUser, err := repository.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	switch callerRole {
	case model.RoleCSAdmin:
		if targetUser.Role != model.RoleCPOP && targetUser.Role != model.RoleCPOM {
			c.JSON(http.StatusForbidden, gin.H{"error": "CS_Admin 只能删除 CP_OP 或 CP_OM 用户"})
			return
		}
	case model.RoleCPOP:
		if targetUser.Role != model.RoleCPOM || targetUser.ParentID == nil || *targetUser.ParentID != callerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "CP_OP 只能删除自己的 CP_OM 用户"})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	if err := repository.DeleteUser(targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
