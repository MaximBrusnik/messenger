package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTUtils interface {
	GenerateToken(userID uint) (string, error)
	ValidateToken(tokenString string) (uint, error)
}

type jwtUtils struct {
	secretKey []byte
}

func NewJWTUtils(secretKey string) JWTUtils {
	return &jwtUtils{
		secretKey: []byte(secretKey),
	}
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func (j *jwtUtils) GenerateToken(userID uint) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "myapp",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *jwtUtils) ValidateToken(tokenString string) (uint, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return 0, errors.New("неверная подпись токена")
		}
		return 0, errors.New("неверный токен")
	}

	if !token.Valid {
		return 0, errors.New("токен невалиден")
	}

	return claims.UserID, nil
}
