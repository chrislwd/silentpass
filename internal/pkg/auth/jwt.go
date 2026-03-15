package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	SessionID  string `json:"session_id"`
	TenantID   string `json:"tenant_id"`
	PhoneHash  string `json:"phone_hash"`
	UseCase    string `json:"use_case"`
	Method     string `json:"method"`
	VerifiedAt int64  `json:"verified_at"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

func NewTokenService(secret string, expiration time.Duration) *TokenService {
	if expiration == 0 {
		expiration = 5 * time.Minute
	}
	return &TokenService{
		secret:     []byte(secret),
		issuer:     "silentpass",
		expiration: expiration,
	}
}

func (s *TokenService) Generate(sessionID, tenantID, phoneHash, useCase, method string) (string, error) {
	now := time.Now()
	claims := &TokenClaims{
		SessionID:  sessionID,
		TenantID:   tenantID,
		PhoneHash:  phoneHash,
		UseCase:    useCase,
		Method:     method,
		VerifiedAt: now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   phoneHash,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) Validate(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
