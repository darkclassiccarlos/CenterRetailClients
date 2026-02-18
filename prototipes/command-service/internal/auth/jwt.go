package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey []byte
	logger    *zap.Logger
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, logger *zap.Logger) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		logger:    logger,
	}
}

// GenerateToken generates a new JWT token with 10 minutes expiration
func (j *JWTManager) GenerateToken(username string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute) // 10 minutes expiration

	claims := JWTClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "command-service",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		j.logger.Error("Failed to generate token", zap.Error(err))
		return "", err
	}

	j.logger.Info("Token generated",
		zap.String("username", username),
		zap.Time("expires_at", expiresAt),
	)

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			j.logger.Warn("Token expired", zap.Error(err))
			return nil, ErrExpiredToken
		}
		j.logger.Warn("Invalid token", zap.Error(err))
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		j.logger.Warn("Invalid token claims")
		return nil, ErrInvalidToken
	}

	return claims, nil
}

