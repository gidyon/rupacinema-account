package service

import (
	"context"
	"database/sql"
	"encoding/json"
	grpc_middleware "github.com/gidyon/rupacinema/account/internal/protocol/grpc/middleware"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"strings"
	"time"
)

type loginDS struct {
	res *account.LoginResponse
	err error
}

func (login *loginDS) Login(
	ctx context.Context, loginReq *account.LoginRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	email := ""
	phone := ""
	hashedPassword := ""
	query := ""

	switch {
	case loginReq.GetFacebook() != nil:
		email = loginReq.GetFacebook().GetEmailAddress()
		phone = loginReq.GetFacebook().GetPhoneNumber()
	case loginReq.GetGoogle() != nil:
		email = loginReq.GetGoogle().GetEmailAddress()
		phone = loginReq.GetGoogle().GetPhoneNumber()
	case loginReq.GetPhone() != nil:
		phone = loginReq.GetPhone().GetPhone()
	default:
		login.err = errMissingCredential("Login Credentials")
		return
	}

	profile := &account.Profile{}

	// `first_name` varchar(50) NOT NULL,
	// `last_name` varchar(50) NOT NULL,
	// `email` varchar(50) DEFAULT NULL,
	// `phone` varchar(15) NOT NULL,
	// `birth_date` date DEFAULT NULL,
	// `gender` enum('male','female','all') NOT NULL DEFAULT 'all',
	// `state` tinyint(1) NOT NULL DEFAULT '1',
	// `security_question` varchar(50) DEFAULT NULL,
	// `security_answer` varchar(40) DEFAULT NULL,
	// `password` text NOT NULL

	// Prepare query
	query = "SELECT first_name, last_name, email, phone, birth_date, gender, state, password FROM users WHERE email=? OR phone=?"

	state := 0
	// Execute query
	row := db.QueryRowContext(ctx, query, email, phone)
	err := row.Scan(
		&profile.FirstName,
		&profile.LastName,
		&profile.EmailAddress,
		&profile.PhoneNumber,
		&profile.BirthDate,
		&profile.Gender,
		&state,
		&hashedPassword,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			login.err = errAccountDoesntExist()
		default:
			login.err = errQueryFailed(err, "GetProfile (SELECT)")
		}
		return
	}

	// Check if password match if they logged in with Email
	if loginReq.GetPhone() != nil {
		err = compareHashedPassword(hashedPassword, loginReq.GetPhone().GetPassword())
		if err != nil {
			login.err = errWrongPassword()
			return
		}
	}

	// Generates the token with claims from profile object
	token, err := grpc_middleware.GenToken(ctx, &account.Profile{
		FirstName:  profile.FirstName,
		LastName:   profile.LastName,
		BirthDate:  time.Now().String(),
		ProfileUrl: time.Now().String(),
	}, &account.Admin{
		UserName: time.Now().String(),
	})
	if err != nil {
		login.err = errFailedToGenToken(err)
		return
	}

	login.res = &account.LoginResponse{
		Token: token,
	}
}

type loginAdminDS struct {
	res *account.LoginResponse
	err error
}

func (login *loginAdminDS) Login(
	ctx context.Context, loginReq *account.LoginAdminRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	userName := loginReq.GetUsername()
	password := loginReq.GetPassword()

	// Check that the request has necessary credential
	err := func() error {
		var err error
		switch {
		case strings.Trim(userName, " ") == "":
			login.err = errMissingCredential("username")
		case strings.Trim(password, " ") == "":
			login.err = errMissingCredential("password")
		}
		return err
	}()

	if err != nil {
		login.err = err
		return
	}

	adminLevel := ""
	adminTrustedDevices := make([]byte, 0)
	passwordHashed := ""

	admin := &account.Admin{}

	// 	`first_name` varchar(50) NOT NULL,
	//  `last_name` varchar(50) NOT NULL,
	//  `email` varchar(50) NOT NULL,
	//  `phone` varchar(15) NOT NULL,
	//  `user_name` varchar(50) NOT NULL,
	//  `admin_level` enum('READER','READER_AND_WRITE_ONLY_FOOD','SUPER_ADMIN') NOT NULL DEFAULT 'READER',
	//  `trusted_devices` json NOT NULL,
	//  `password` text NOT NULL

	// Prepare query
	query := "SELECT * FROM admins WHERE user_name=?"

	// Execute query
	row := db.QueryRowContext(ctx, query, userName)
	err = row.Scan(
		&admin.FirstName,
		&admin.LastName,
		&admin.EmailAddress,
		&admin.PhoneNumber,
		&admin.UserName,
		&adminLevel,
		&adminTrustedDevices,
		&passwordHashed,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			login.err = errAccountDoesntExist()
		default:
			login.err = errQueryFailed(err, "GetProfile (SELECT)")
		}
		return
	}

	if err = json.Unmarshal(adminTrustedDevices, &admin.TrustedDevices); err != nil {
		login.err = errFromJSONUnMarshal(err, "Admin.TrustedDevices")
		return
	}

	admin.Level = account.AdminLevel(account.AdminLevel_value[adminLevel])

	// Check if password match if they logged in with Email
	err = compareHashedPassword(passwordHashed, password)
	if err != nil {
		login.err = errWrongPassword()
		return
	}

	// Generates the token with claims from admin object
	token, err := grpc_middleware.GenToken(ctx, &account.Profile{
		BirthDate:  time.Now().String(),
		ProfileUrl: time.Now().String(),
	}, &account.Admin{
		FirstName: admin.FirstName,
		LastName:  admin.LastName,
		UserName:  time.Now().String(),
	})
	if err != nil {
		login.err = errFailedToGenToken(err)
		return
	}

	login.res = &account.LoginResponse{
		Token: token,
	}
}
