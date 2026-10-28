package auth

import (
	"errors"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultAuthTokenTTL  = 24 * time.Hour
	rememberAuthTokenTTL = 7 * 24 * time.Hour
)

// AuthClaims are the claims of a sign-in JWT. Subject is the user's email.
type AuthClaims struct {
	UID        string `json:"uid"`
	RememberMe bool   `json:"rmb"`
	jwt.RegisteredClaims
}

// NewAuthClaims returns claims for user that expire after 24h, or 7 days with
// rememberMe.
func NewAuthClaims(user models.User, rememberMe bool) AuthClaims {
	return AuthClaims{
		UID:        user.UID,
		RememberMe: rememberMe,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(authTokenTTL(rememberMe))),
		},
	}
}

// JWTCreateToken returns a JWT token string for claims, optionally error
func JWTCreateToken(env *appenv.Env, claims AuthClaims) (string, error) {
	secret, err := jwtSecretFromEnv(env)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// RefreshJWTToken returns a refreshed JWT token for claims if expiring in less than 1h and still valid, optionally error
func RefreshJWTToken(env *appenv.Env, token string, claims AuthClaims) (string, time.Duration, error) {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return "", 0, err
	}
	if exp == nil {
		return "", 0, errors.New("parse JWT token claims, exp not found")
	}

	if time.Until(exp.Time) > time.Hour {
		return token, time.Until(exp.Time), nil
	}

	validFor := authTokenTTL(claims.RememberMe)
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(validFor))

	refreshed, err := JWTCreateToken(env, claims)
	if err != nil {
		return "", 0, err
	}

	return refreshed, validFor, nil
}

// decodeJWTToken returns the claims of a valid, unexpired JWT token
func decodeJWTToken(env *appenv.Env, tokenStr string) (AuthClaims, error) {
	secret, err := jwtSecretFromEnv(env)
	if err != nil {
		return AuthClaims{}, err
	}

	claims := AuthClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", errors.New("parse JWT token")
		}

		return []byte(secret), nil
	})
	if err != nil {
		return AuthClaims{}, err
	}

	if claims.UID == "" {
		return AuthClaims{}, errors.New("parse JWT token claims, uid not found")
	}

	return claims, nil
}

func authTokenTTL(rememberMe bool) time.Duration {
	if rememberMe {
		return rememberAuthTokenTTL
	}

	return defaultAuthTokenTTL
}

func jwtSecretFromEnv(env *appenv.Env) (string, error) {
	if env == nil || env.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	return env.JWTSecret, nil
}
