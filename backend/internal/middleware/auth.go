package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/auth"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/mylog"
)

const (
	CtxUserID   = "userID"
	CtxUsername = "username"
	CtxRole     = "role"
	CtxClaims   = "claims"
	CtxTenantID = "tenantID"
)

// JWT validates Bearer token and injects claims into context
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, string(claims.Role))
		c.Set(CtxClaims, claims)
		// Inject tenant_id for tenant-scoped users (CP_OP, CP_OM)
		if claims.Role.TenantScoped() && claims.TenantID != "" {
			c.Set(CtxTenantID, claims.TenantID)
		}
		c.Next()
	}
}

// RequireRole aborts if the caller doesn't have one of the allowed roles
func RequireRole(roles ...model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := model.Role(c.GetString(CtxRole))
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// RequireCSAdminOrCPOP allows CS_Admin and CP_OP
func RequireCSAdminOrCPOP() gin.HandlerFunc {
	return RequireRole(model.RoleCSAdmin, model.RoleCPOP)
}

// RequireCSAdmin allows CS_Admin only
func RequireCSAdmin() gin.HandlerFunc {
	return RequireRole(model.RoleCSAdmin)
}

// Logger logs every HTTP request to the mylog package.
// Only logs API paths (/api/*, /health, /firmware) to avoid filling
// event logs with scanner noise (/wiki, /login.asp, /, etc.).
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(CtxUserID)
		if userID == "" {
			userID = "anonymous"
		}
		role := c.GetString(CtxRole)
		if role == "" {
			role = "-"
		}
		c.Next()
		// Skip WebSocket and non-API paths (scanner probes, etc.)
		p := c.Request.URL.Path
		if p == "/api/events/ws" {
			return
		}
		if !strings.HasPrefix(p, "/api/") && p != "/health" && !strings.HasPrefix(p, "/firmware") {
			return
		}
		mylog.Write(userID, "[HTTP] %s %s %s %d", role, c.Request.Method, p, c.Writer.Status())
	}
}

// RequireCPOP allows CP_OP only
func RequireCPOP() gin.HandlerFunc {
	return RequireRole(model.RoleCPOP)
}

// CORS adds permissive CORS headers for development; tighten in production
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
