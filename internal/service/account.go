package service

import (
	"context"
	"database/sql"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/movie/pkg/logger"
	"github.com/gidyon/rupacinema/notification/pkg/api"
	"github.com/golang/protobuf/ptypes/empty"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accountAPIServer struct {
	ctx                       context.Context
	db                        *sql.DB
	sqlWorkerChan             chan sqlWorker
	notificationServiceClient notification.NotificationServiceClient
}

type sqlWorker struct {
	query string
	args  []interface{}
	err   error
}

// NewAccountAPIServer is a AccountAPIServer
func NewAccountAPIServer(
	ctx context.Context,
	db *sql.DB,
	notificationServiceClient notification.NotificationServiceClient,
) (account.AccountAPIServer, error) {
	return &accountAPIServer{
		ctx:                       ctx,
		db:                        db,
		sqlWorkerChan:             make(chan sqlWorker, 0),
		notificationServiceClient: notificationServiceClient,
	}, nil
}

func (accountAPISrv *accountAPIServer) Login(
	ctx context.Context, loginReq *account.LoginRequest,
) (*account.LoginResponse, error) {
	ctxLogin, cancel := context.WithCancel(ctx)
	defer cancel()

	login := &loginDS{}

	login.Login(ctxLogin, loginReq, accountAPISrv.db)

	if cancelled(ctxLogin) {
		login.err = contextError(ctxLogin, "Login")
	}

	return login.res, login.err
}

func (accountAPISrv *accountAPIServer) CreateUser(
	ctx context.Context, createReq *account.CreateUserRequest,
) (*empty.Empty, error) {
	ctxCreate, cancel := context.WithCancel(ctx)
	defer cancel()

	createUser := &createUserDS{}

	createUser.Create(ctxCreate, accountAPISrv.sqlWorkerChan, createReq, accountAPISrv.db)

	if cancelled(ctxCreate) {
		createUser.err = contextError(ctxCreate, "Create")
	}

	return createUser.res, createUser.err
}

func (accountAPISrv *accountAPIServer) GetUser(
	ctx context.Context, getReq *account.GetUserRequest,
) (*account.Profile, error) {
	ctxGet, cancel := context.WithCancel(ctx)
	defer cancel()

	getProfile := &getProfileDS{}

	getProfile.Get(ctxGet, getReq, accountAPISrv.db)

	if cancelled(ctxGet) {
		getProfile.err = contextError(ctxGet, "Create")
	}

	return getProfile.res, getProfile.err
}

func (accountAPISrv *accountAPIServer) AuthenticateUser(
	ctx context.Context, authReq *account.AuthenticateUserRequest,
) (*account.AuthenticateResponse, error) {
	ctxAuth, cancel := context.WithCancel(ctx)
	defer cancel()

	authAccount := &authAccountDS{}

	authAccount.Authenticate(ctxAuth, authReq, accountAPISrv.db)

	if cancelled(ctxAuth) {
		authAccount.err = contextError(ctxAuth, "Authenticate")

	}

	return authAccount.res, authAccount.err
}

func (accountAPISrv *accountAPIServer) ListUsers(
	listReq *account.ListUsersRequest, listSrv account.AccountAPI_ListUsersServer,
) error {
	// Prepare query
	query := `SELECT first_name, last_name, email, phone, birth_date FROM profiles`
	// Execute query
	rows, err := accountAPISrv.db.Query(query)
	if err != nil {
		return errQueryFailed(err, "GETUsers (SELECT)")
	}

	// Send it via a stream
	for rows.Next() {
		profile := &account.Profile{}
		err = rows.Scan(profile.FirstName, profile.LastName, profile.EmailAddress, profile.BirthDate)
		if err != nil {
			logger.Log.Error("error scanning results", zap.Error(err))
			continue
		}
		err = listSrv.Send(profile)
		if err != nil {
			logger.Log.Error("error sending results", zap.Error(err))
			continue
		}
	}

	return nil
}

func (accountAPISrv *accountAPIServer) LoginAdmin(
	ctx context.Context, loginReq *account.LoginAdminRequest,
) (*account.LoginResponse, error) {
	ctxLogin, cancel := context.WithCancel(ctx)
	defer cancel()

	login := &loginAdminDS{}

	login.Login(ctxLogin, loginReq, accountAPISrv.db)

	if cancelled(ctxLogin) {
		login.err = contextError(ctxLogin, "Login")
	}

	return login.res, login.err
}

func (accountAPISrv *accountAPIServer) CreateAdmin(
	ctx context.Context, createReq *account.CreateAdminRequest,
) (*empty.Empty, error) {
	ctxCreate, cancel := context.WithCancel(ctx)
	defer cancel()

	createAdmin := &createAdminDS{}

	createAdmin.Create(ctxCreate, accountAPISrv.sqlWorkerChan, createReq, accountAPISrv.db)

	if cancelled(ctxCreate) {
		createAdmin.err = contextError(ctxCreate, "Create")
	}

	return createAdmin.res, createAdmin.err

}
func (accountAPISrv *accountAPIServer) AuthenticateAdmin(
	ctx context.Context, authReq *account.AuthenticateAdminRequest,
) (*account.AuthenticateResponse, error) {
	ctxAuth, cancel := context.WithCancel(ctx)
	defer cancel()

	authAdmin := &authAdminDS{}

	authAdmin.Authenticate(ctxAuth, authReq, accountAPISrv.db)

	if cancelled(ctxAuth) {
		authAdmin.err = contextError(ctxAuth, "Authenticate")

	}

	return authAdmin.res, authAdmin.err
}

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
