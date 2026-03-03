package auth

import (
	"errors"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/golang-jwt/jwt/v5"
)

// JWTCreateToken returns a JWT token string for the sub and uid, optionally error
func JWTCreateToken(env *appenv.Env, sub, uid string) (string, error) {
	secret, err := jwtSecretFromEnv(env)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": sub,
		"uid": uid,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
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
func RefreshJWTToken(env *appenv.Env, token string, claims jwt.MapClaims) (string, error) {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return "", err
	}

	if time.Until(exp.Time) > time.Hour {
		return token, nil
	}

	email, err := claims.GetSubject()
	if err != nil {
		return "", err
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("failed to parse JWT token claims, uid not found")
	}

	jwt, err := JWTCreateToken(env, email, uid)
	if err != nil {
		return "", err
	}

	return jwt, nil
}

func jwtSecretFromEnv(env *appenv.Env) (string, error) {
	if env == nil || env.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	return env.JWTSecret, nil
}
