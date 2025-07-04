package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sh5080/ndns-router/pkg/configs"
)

var jwtSecret = []byte(configs.GetConfig().App.JwtSecret)

// JwtClaims 구조체
type SseClaims struct {
	ReqId string `json:"reqId"`
	jwt.RegisteredClaims
}

// Jwt 생성 함수
func GenerateSseToken(reqId string, ttlMinutes int) (string, error) {
	claims := SseClaims{
		ReqId: reqId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(ttlMinutes))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Jwt 검증 함수
func ParseAndValidateSseToken(tokenStr string) (*SseClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &SseClaims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*SseClaims)
	if !ok {
		return nil, err
	}

	return claims, nil
}
