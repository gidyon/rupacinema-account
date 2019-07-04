package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"strings"
)

type getProfileDS struct {
	res *account.Profile
	err error
}

func (getProfile *getProfileDS) Get(
	ctx context.Context, getReq *account.GetUserRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	email := getReq.GetEmail()
	phoneNumber := getReq.GetPhone()

	// Check that the request has necessary credential
	if strings.Trim(email, " ") == "" && strings.Trim(phoneNumber, " ") == "" {
		getProfile.err = errMissingCredential("email and phone")
		return
	}

	profile, err := getProfileFromDB(ctx, db, email, phoneNumber)
	if err != nil {
		getProfile.err = err
		return
	}

	getProfile.res = profile
}

func getProfileFromDB(
	ctx context.Context, db *sql.DB, email, phoneNumber string,
) (*account.Profile, error) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

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
	query := `SELECT first_name, last_name, email, phone, birth_date, gender, state FROM users WHERE email=? OR phone=?`

	// Execute query
	row := db.QueryRowContext(ctx, query, email, phoneNumber)

	profile := &account.Profile{}
	state := 0

	err := row.Scan(
		&profile.FirstName,
		&profile.LastName,
		&profile.EmailAddress,
		&profile.PhoneNumber,
		&profile.BirthDate,
		&profile.Gender,
		&state,
	)
	if err != nil {
		return nil, errQueryFailed(err, "GetProfile (SELECT)")
	}

	if state != 1 {
		return nil, errAccountBlocked()
	}

	return profile, nil
}

type getAdminDS struct {
	res *account.Admin
	err error
}

func (getAdmin *getAdminDS) Get(
	ctx context.Context, getReq *account.GetAdminRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	userName := getReq.GetUserName()

	// Check that the request has necessary credential
	if strings.Trim(userName, " ") == "" {
		getAdmin.err = errMissingCredential("user name")
		return
	}

	admin, err := getAdminFromDB(ctx, db, userName)
	if err != nil {
		getAdmin.err = err
		return
	}

	getAdmin.res = admin
}

func getAdminFromDB(
	ctx context.Context, db *sql.DB, userName string,
) (*account.Admin, error) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	// 	`first_name` varchar(50) NOT NULL,
	//  `last_name` varchar(50) NOT NULL,
	//  `email` varchar(50) NOT NULL,
	//  `phone` varchar(15) NOT NULL,
	//  `user_name` varchar(50) NOT NULL,
	//  `admin_level` enum('READER','READER_AND_WRITE_ONLY_FOOD','SUPER_ADMIN') NOT NULL DEFAULT 'READER',
	//  `trusted_devices` json NOT NULL,
	//  `password` text NOT NULL

	// Prepare query
	query := `SELECT * FROM admins WHERE user_name=?`

	// Execute query
	row := db.QueryRowContext(ctx, query, userName)

	admin := &account.Admin{}
	level := ""
	trustedDevices := make([]byte, 0)
	password := ""

	err := row.Scan(
		&admin.FirstName,
		&admin.LastName,
		&admin.EmailAddress,
		&admin.PhoneNumber,
		&admin.UserName,
		&level,
		&trustedDevices,
		&password,
	)
	if err != nil {
		return nil, errQueryFailed(err, "GetAdmin (SELECT)")
	}

	err = json.Unmarshal(trustedDevices, &admin.TrustedDevices)
	if err != nil {
		return nil, errFromJSONUnMarshal(err, "Admin.TrustedDevices")
	}

	return admin, nil
}
