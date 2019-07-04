package service

import (
	"context"
	"database/sql"
	"github.com/gidyon/rupacinema/account/internal/protocol/grpc/middleware"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/movie/pkg/logger"
	"github.com/gidyon/rupacinema/notification/pkg/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strings"
	"time"
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

	createUser.Create(
		ctxCreate,
		accountAPISrv.sqlWorkerChan,
		createReq,
		accountAPISrv.db,
		accountAPISrv.notificationServiceClient,
	)

	if cancelled(ctxCreate) {
		createUser.err = contextError(ctxCreate, "CreateUser")
	}

	return createUser.res, createUser.err
}

func (accountAPISrv *accountAPIServer) GetDefaultToken(
	ctx context.Context, createReq *account.GetDefaultTokenRequest,
) (*account.LoginResponse, error) {
	token, err := middleware.GenToken(ctx, &account.Profile{
		FirstName: uuid.New().String(),
		LastName:  uuid.New().String(),
		BirthDate: time.Now().String(),
	}, &account.Admin{
		FirstName: uuid.New().String(),
		LastName:  uuid.New().String(),
		UserName:  time.Now().String(),
	})
	if err != nil {
		return nil, err
	}
	return &account.LoginResponse{
		Token: token,
	}, nil
}

func (accountAPISrv *accountAPIServer) AuthenticateRequest(
	ctx context.Context, _ *empty.Empty,
) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (accountAPISrv *accountAPIServer) GetUser(
	ctx context.Context, getReq *account.GetUserRequest,
) (*account.Profile, error) {
	ctxGet, cancel := context.WithCancel(ctx)
	defer cancel()

	getProfile := &getProfileDS{}

	getProfile.Get(ctxGet, getReq, accountAPISrv.db)

	if cancelled(ctxGet) {
		getProfile.err = contextError(ctxGet, "GetUser")
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
		authAccount.err = contextError(ctxAuth, "AuthenticateUser")

	}

	return authAccount.res, authAccount.err
}

func (accountAPISrv *accountAPIServer) ListUsers(
	listReq *account.ListUsersRequest, listSrv account.AccountAPI_ListUsersServer,
) error {
	// Prepare query
	query := `SELECT first_name, last_name, email, phone, birth_date FROM users`
	// Execute query
	rows, err := accountAPISrv.db.Query(query)
	if err != nil {
		return errQueryFailed(err, "ListUsers (SELECT)")
	}

	// Send it via a stream
	for rows.Next() {
		profile := &account.Profile{}
		err = rows.Scan(
			&profile.FirstName,
			&profile.LastName,
			&profile.EmailAddress,
			&profile.BirthDate,
		)
		if err != nil {
			logger.Log.Error("error scanning results", zap.Error(err))
			return rows.Err()
		}
		err = listSrv.Send(profile)
		if err != nil {
			logger.Log.Error("error sending results", zap.Error(err))
			return rows.Err()
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
		login.err = contextError(ctxLogin, "LoginAdmin")
	}

	return login.res, login.err
}

func (accountAPISrv *accountAPIServer) CreateAdmin(
	ctx context.Context, createReq *account.CreateAdminRequest,
) (*empty.Empty, error) {

	// Validate super admin username is provided
	superAdminName := createReq.GetSuperAdminUsername()
	if strings.Trim(superAdminName, " ") == "" {
		return nil, errMissingCredential("SuperAdminUsername")
	}

	// Authenticate the superadmin
	level, err := checkAdminInDB(ctx, accountAPISrv.db, superAdminName)
	if err != nil {
		return nil, err
	}
	if level != account.AdminLevel_SUPER_ADMIN {
		return nil, errPermissionDenied("CreateAdmin")
	}

	ctxCreate, cancel := context.WithCancel(ctx)
	defer cancel()

	createAdmin := &createAdminDS{}

	createAdmin.Create(
		ctxCreate,
		accountAPISrv.sqlWorkerChan,
		createReq,
		accountAPISrv.db,
		accountAPISrv.notificationServiceClient,
	)

	if cancelled(ctxCreate) {
		createAdmin.err = contextError(ctxCreate, "CreateAdmin")
	}

	return createAdmin.res, createAdmin.err

}

func (accountAPISrv *accountAPIServer) GetAdmin(
	ctx context.Context, getReq *account.GetAdminRequest,
) (*account.Admin, error) {
	ctxGet, cancel := context.WithCancel(ctx)
	defer cancel()

	getAdmin := &getAdminDS{}

	getAdmin.Get(ctxGet, getReq, accountAPISrv.db)

	if cancelled(ctxGet) {
		getAdmin.err = contextError(ctxGet, "GetAdmin")
	}

	return getAdmin.res, getAdmin.err
}

func (accountAPISrv *accountAPIServer) AuthenticateAdmin(
	ctx context.Context, authReq *account.AuthenticateAdminRequest,
) (*account.AuthenticateResponse, error) {
	ctxAuth, cancel := context.WithCancel(ctx)
	defer cancel()

	authAdmin := &authAdminDS{}

	authAdmin.Authenticate(ctxAuth, authReq, accountAPISrv.db)

	if cancelled(ctxAuth) {
		authAdmin.err = contextError(ctxAuth, "AuthenticateAdmin")

	}

	return authAdmin.res, authAdmin.err
}
