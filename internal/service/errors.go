package service

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checks whether a given context has been cancelled
func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

// contextError wraps context error to a gRPC error
func contextError(ctx context.Context, operation string) error {
	if _, ok := ctx.Err().(interface{ Timeout() bool }); ok {
		// Should retry the request
		return status.Errorf(codes.DeadlineExceeded, "couldn't complete %s operation: %v", operation, ctx.Err())
	}
	return status.Errorf(codes.Canceled, "couldn't complete %s operation: %v", operation, ctx.Err())
}

func errFromJSONMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to json marshal %s: %v", obj, err)
}

func errFromJSONUnMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to json unmarshal %s: %v", obj, err)
}

func errQueryFailed(err error, queryType string) error {
	return status.Errorf(codes.Internal, "failed to execute %s query: %v", queryType, err)
}

func errQueryNoRows(err error) error {
	return status.Errorf(codes.NotFound, "no rows found for query: %v", err)
}

func errMissingCredential(cred string) error {
	return status.Errorf(codes.FailedPrecondition, "missing credentials: %v", cred)
}

func errCheckingCreds(err error) error {
	return status.Errorf(codes.Internal, "failed while checking credentials: %v", err)
}

func errPermissionDenied(op string) error {
	return status.Errorf(codes.PermissionDenied, "not authorised to perform %s operation", op)
}

func errFailedToGenToken(err error) error {
	return status.Errorf(codes.Internal, "failed to generate jwt token: %v", err)
}

func errAccountBlocked() error {
	return status.Error(codes.PermissionDenied, "account has been blocked - contact sysadmin")
}

func errAccountDoesntExist() error {
	return status.Error(codes.NotFound, "account does not exist")
}

func errAccountDoesExist() error {
	return status.Error(codes.NotFound, "account exists")
}

func errWrongPassword() error {
	return status.Error(codes.Unauthenticated, "wrong password")
}

func errFailedToGenHashedPass(err error) error {
	return status.Errorf(codes.Internal, "failed to generate hashed password: %v", err)
}
