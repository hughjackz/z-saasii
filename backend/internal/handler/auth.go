package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/auth"
	"github.com/yourorg/csms-backend/internal/middleware"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

// POST /api/auth/login
func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}

	user, err := repository.GetUserByUsername(req.Username)
	if err != nil || !user.Enabled {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !repository.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{
		Token: token,
		User:  repository.ToUserResponse(user),
	})
}

// POST /api/auth/logout  (stateless JWT — just acknowledge)
func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// GET /api/auth/me
func Me(c *gin.Context) {
	userID := c.GetString(middleware.CtxUserID)
	user, err := repository.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, repository.ToUserResponse(user))
}
