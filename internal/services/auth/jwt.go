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

// needsRefresh reports whether less than half of the token's lifetime is left.
func (ac AuthClaims) needsRefresh(now time.Time) bool {
	if ac.ExpiresAt == nil {
		return true
	}

	return ac.ExpiresAt.Sub(now) < authTokenTTL(ac.RememberMe)/2
}

// renewed returns the claims with a full lifetime from now.
func (ac AuthClaims) renewed(now time.Time) AuthClaims {
	ac.ExpiresAt = jwt.NewNumericDate(now.Add(authTokenTTL(ac.RememberMe)))
	return ac
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
