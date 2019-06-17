package service

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/gidyon/rupacinema/account/pkg/api"
)

var signingKey = []byte("MySecretSoupRecipeOrAvengersEndGame")

// JWTClaims contains jwt claims
type JWTClaims struct {
	*account.Profile
	*account.Admin
	jwt.StandardClaims
}

// generates json web token
func genToken(ctx context.Context, profile *account.Profile, admin *account.Admin) (string, error) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return "", ctx.Err()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		profile,
		admin,
		jwt.StandardClaims{
			ExpiresAt: 1500,
			Issuer:    "Rupa Cinema",
		},
	})

	// Generate the token ...
	return token.SignedString(signingKey)
}
