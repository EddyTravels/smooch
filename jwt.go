package smooch

import (
	jwt "github.com/dgrijalva/jwt-go"
)

func GenerateJWT(scope string, keyID string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"scope": scope,
	})
	token.Header = map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
		"kid": keyID,
	}

	return token.SignedString([]byte(secret))
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
