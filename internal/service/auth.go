package service

import (
	"context"
	"database/sql"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"strings"
)

type authAccountDS struct {
	res *account.AuthenticateResponse
	err error
}

// Authenticate checks the credentials of an account if it matches the records in the db
func (authAccount *authAccountDS) Authenticate(
	ctx context.Context, authReq *account.AuthenticateUserRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	email := authReq.GetEmail()
	phoneNumber := authReq.GetPhone()

	// Check that the request has necessary credentials
	if strings.Trim(email, " ") == "" && strings.Trim(phoneNumber, " ") == "" {
		authAccount.err = errMissingCredential("Email And Phone")
		return
	}

	firstName := ""
	// Query db to check if an account exist with provided credentials
	query := `SELECT first_name FROM users WHERE email=? OR phone=?`
	// Execute query
	row := db.QueryRowContext(ctx, query, email, phoneNumber)

	err := row.Scan(&firstName)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			authAccount.err = errAccountDoesntExist()
		default:
			authAccount.err = errQueryFailed(err, "AuthenticateUser (SELECT)")
		}
		return
	}

	authAccount.res = &account.AuthenticateResponse{
		Valid: true,
	}
}

type authAdminDS struct {
	res *account.AuthenticateResponse
	err error
}

func (authAdmin *authAdminDS) Authenticate(
	ctx context.Context, authReq *account.AuthenticateAdminRequest, db *sql.DB,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	userName := authReq.GetUsername()

	// Check that the request has necessary credentials
	if strings.Trim(userName, " ") == "" {
		authAdmin.err = errMissingCredential("user name")
		return
	}

	level, err := checkAdminInDB(ctx, db, userName)
	if err != nil {
		authAdmin.err = err
		return
	}

	valid := true
	if level != authReq.GetLevel() {
		valid = false
	}

	authAdmin.res = &account.AuthenticateResponse{
		Valid: valid,
	}
}

func checkAdminInDB(
	ctx context.Context, db *sql.DB, userName string,
) (account.AdminLevel, error) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return -1, ctx.Err()
	}

	level := ""
	// Query db to check if an account exist with provided credentials
	query := `SELECT admin_level FROM admins WHERE user_name=?`
	// Execute query
	row := db.QueryRowContext(ctx, query, userName)

	err := row.Scan(&level)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return -1, errAccountDoesntExist()
	default:
		return -1, errQueryFailed(err, "AuthenticateAdmin (SELECT)")
	}

	return account.AdminLevel(account.AdminLevel_value[level]), nil
}
