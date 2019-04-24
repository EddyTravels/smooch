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
