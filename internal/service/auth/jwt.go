package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTCreateToken returns a JWT token string for the sub and uid, optionally error
func JWTCreateToken(sub, uid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": sub,
		"uid": uid,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// decodeJWTToken returns the uid string from the JWT claims
func decodeJWTToken(tokenStr string) (claims jwt.MapClaims, err error) {
	secret := os.Getenv("JWT_SECRET")
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
func RefreshJWTToken(token string, claims jwt.MapClaims) (string, error) {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return "", err
	}

	t1 := exp.Time
	t2 := time.Now()
	diff := t2.Sub(t1)
	if diff > time.Hour {
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

	jwt, err := JWTCreateToken(email, uid)
	if err != nil {
		return "", err
	}

	return jwt, nil
}
