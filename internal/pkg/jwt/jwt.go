package jwt

import (
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims JWT 载荷，包含 user_id 和 username
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`  // 用户唯一标识
	Username string    `json:"username"` // 用户名
	gojwt.RegisteredClaims
}

// Generate 签发 JWT Access Token，使用 HS256 算法
// ttl 为有效期，例如 24 * time.Hour
func Generate(userID uuid.UUID, username string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  gojwt.NewNumericDate(now),
			Issuer:    "cloudemu",
		},
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// Parse 解析并验证 JWT Token，返回 Claims
func Parse(tokenStr string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	_, err := gojwt.ParseWithClaims(tokenStr, claims,
		func(t *gojwt.Token) (interface{}, error) { return secret, nil },
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
