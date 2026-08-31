package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/csms-backend/config"
	"github.com/yourorg/csms-backend/internal/model"
)

type Claims struct {
	UserID   string     `json:"userId"`
	Username string     `json:"username"`
	Role     model.Role `json:"role"`
	TenantID string     `json:"tenantId"` // CP_OP.id for tenant-scoped users, empty for CS_Admin
	jwt.RegisteredClaims
}

func GenerateToken(user *model.User) (string, error) {
	exp := time.Duration(config.Cfg.JWT.ExpireHours) * time.Hour
	tenantID := ""
	if user.TenantID != nil {
		tenantID = *user.TenantID
	}
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Cfg.JWT.Secret))
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.Cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
