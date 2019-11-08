package smooch

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// JWTExpiration defines how many seconds jwt token is valid
const JWTExpiration = 3600

func GenerateJWT(scope string, keyID string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"scope": scope,
		"exp":   JWTExpiration,
	})
	token.Header = map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
		"kid": keyID,
	}

	return token.SignedString([]byte(secret))
}

// getJWTExpiration will get jwt expiration time
func getJWTExpiration(jwtToken string, secret string) (int64, error) {
	claims := jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(jwtToken, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return -1, err
	}

	expiredIn := claims["exp"].(int64) - time.Now().Unix()
	return expiredIn, nil
}

// isJWTExpired will check whether Smooch JWT is expired or not.
func isJWTExpired(jwtToken string, secret string) (bool, error) {
	_, err := jwt.ParseWithClaims(jwtToken, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err == nil {
		return false, nil
	}

	switch err.(type) {
	case *jwt.ValidationError:
		vErr := err.(*jwt.ValidationError)
		if vErr.Errors == jwt.ValidationErrorExpired {
			return true, nil
		}
	}
	return false, err
}
