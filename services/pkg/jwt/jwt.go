// Package jwt provides shared JWT generation and validation utilities.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims is the JWT claims structure embedded in all tokens.
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// Manager generates and validates HS256 JWTs.
type Manager struct {
	secret []byte
}

// NewManager creates a JWT Manager with the given secret key.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// GenerateToken issues a signed token for the user id, valid for 24h.
func (m *Manager) GenerateToken(userID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "messengermax",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateToken parses and verifies a token, returning the user id.
func (m *Manager) ValidateToken(tokenString string) (uint, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, errors.New("invalid token")
	}
	return claims.UserID, nil
}
