package middleware

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	signingKey    = []byte("MySecretSoupRecipeOrAvengersEndGame")
	signingMethod jwt.SigningMethod
)

// JWTClaims contains JWT claims information
type JWTClaims struct {
	*account.Profile
	*account.Admin
	jwt.StandardClaims
}

// GenToken json web token
func GenToken(ctx context.Context, profile *account.Profile, admin *account.Admin) (string, error) {
	token := jwt.NewWithClaims(signingMethod, JWTClaims{
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

type tokenKey int

var key tokenKey = 1

// GetTokenInfo returns the value in context
func GetTokenInfo(ctx context.Context) (*AdminAndUserDS, error) {
	val, ok := ctx.Value(key).(*AdminAndUserDS)
	if !ok {
		return nil, fmt.Errorf("failed to cast context val to *AdminAndUserDS")
	}
	return val, nil
}

func parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parde token with claims: %v", err)
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, status.Error(codes.FailedPrecondition, "token must be valid")
	}
	return claims, nil
}

// AdminAndUserDS has admin and user as part of JWTClaims object
type AdminAndUserDS struct {
	admin *account.Admin
	user  *account.Profile
}

func userClaimFromToken(claims *JWTClaims) *AdminAndUserDS {
	return &AdminAndUserDS{
		admin: claims.Admin,
		user:  claims.Profile,
	}
}

// AddAuthentication returns grpc.Server config option that turn on logging.
func AddAuthentication(
	signingKeyP []byte, signingMethodP jwt.SigningMethod,
) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {

	if string(signingKeyP) != "" {
		signingKey = signingKeyP
	}

	signingMethod = signingMethodP

	authFunc := func(ctx context.Context) (context.Context, error) {
		token, err := grpc_auth.AuthFromMD(ctx, "Bearer")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get Bearer: %v", err)
		}

		_, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "metadata missing")
		}

		tokenInfo, err := parseToken(token)
		if err != nil {
			return nil, err
		}

		grpc_ctxtags.Extract(ctx).Set("auth.sub", userClaimFromToken(tokenInfo))

		newCtx := context.WithValue(ctx, key, tokenInfo)

		return newCtx, nil
	}

	return grpc_auth.UnaryServerInterceptor(authFunc), grpc_auth.StreamServerInterceptor(authFunc)
}
