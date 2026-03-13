package auth

import (
	"errors"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultAuthTokenTTL  = 24 * time.Hour
	rememberAuthTokenTTL = 7 * 24 * time.Hour
)

// JWTCreateToken returns a JWT token string for the sub and uid, optionally error
func JWTCreateToken(env *appenv.Env, sub, uid string) (string, error) {
	return JWTCreateTokenWithTTL(env, sub, uid, defaultAuthTokenTTL, false)
}

func JWTCreateTokenWithTTL(env *appenv.Env, sub, uid string, validFor time.Duration, rememberMe bool) (string, error) {
	secret, err := jwtSecretFromEnv(env)
	if err != nil {
		return "", err
	}

	if validFor <= 0 {
		validFor = defaultAuthTokenTTL
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": sub,
		"uid": uid,
		"rmb": rememberMe,
		"exp": time.Now().Add(validFor).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// decodeJWTToken returns the uid string from the JWT claims
func decodeJWTToken(env *appenv.Env, tokenStr string) (claims jwt.MapClaims, err error) {
	secret, err := jwtSecretFromEnv(env)
	if err != nil {
		return jwt.MapClaims{}, err
	}

	tokenJWT, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) { // Don't forget to validate the algorithm is what you expect:
		// Don't forget to validate the algorithm is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", errors.New("failed to jarse JWT token")
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
	if err != nil {
		return jwt.MapClaims{}, err
	}

	claims, ok := tokenJWT.Claims.(jwt.MapClaims)
	if !ok {
		return jwt.MapClaims{}, errors.New("failed to parse JWT token claims")
	}

	uid, ok := claims["uid"].(string)
	if !ok || uid == "" {
		return jwt.MapClaims{}, errors.New("failed to parse JWT token claims, uid not found")
	}

	return claims, nil
}

// RefreshJWTToken returns a refreshed JWT token for the sub and uid if expiring in less than 1h and still valid, optionally error
func RefreshJWTToken(env *appenv.Env, token string, claims jwt.MapClaims) (string, time.Duration, error) {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return "", 0, err
	}

	if time.Until(exp.Time) > time.Hour {
		return token, time.Until(exp.Time), nil
	}

	email, err := claims.GetSubject()
	if err != nil {
		return "", 0, err
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		return "", 0, errors.New("failed to parse JWT token claims, uid not found")
	}

	rememberMe := rememberMeFromClaims(claims)
	validFor := authTokenTTL(rememberMe)

	jwt, err := JWTCreateTokenWithTTL(env, email, uid, validFor, rememberMe)
	if err != nil {
		return "", 0, err
	}

	return jwt, validFor, nil
}

func authTokenTTL(rememberMe bool) time.Duration {
	if rememberMe {
		return rememberAuthTokenTTL
	}

	return defaultAuthTokenTTL
}

func rememberMeFromClaims(claims jwt.MapClaims) bool {
	rememberMe, ok := claims["rmb"].(bool)
	if ok {
		return rememberMe
	}

	return false
}

func jwtSecretFromEnv(env *appenv.Env) (string, error) {
	if env == nil || env.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	return env.JWTSecret, nil
}
