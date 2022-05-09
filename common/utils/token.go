package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type TokenData struct {
	Data map[string]interface{} `json:"data"`
	jwt.StandardClaims
}

// CreateJwtToken jwt 创建token
func CreateJwtToken(info map[string]interface{}, secret string, expiration uint) (string, error) {
	exp := time.Now().Add(time.Hour * time.Duration(expiration)).Unix()
	claims := TokenData{
		Data: info,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
			Issuer:    info["issuer"].(string),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// AccessJwtToken jwt 验证
func AccessJwtToken(tokenString, secret string) (data map[string]interface{}, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenData{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return
	}
	if claims, ok := token.Claims.(*TokenData); ok && token.Valid {
		return claims.Data, nil
	}
	return nil, fmt.Errorf("Invalid token")
}
