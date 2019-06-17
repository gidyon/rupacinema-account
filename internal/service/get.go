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

func getProfileFromDB(ctx context.Context, db *sql.DB, email, phoneNumber string) (*account.Profile, error) {
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
	// `notification_method` enum('EMAIL_AND_PHONE','EMAIL_ONLY','PHONE_ONLY') NOT NULL DEFAULT 'EMAIL_AND_PHONE',
	// `subscribed_notifications` json NOT NULL,
	// `state` tinyint(1) NOT NULL DEFAULT '1',
	// `security_question` varchar(50) DEFAULT NULL,
	// `security_answer` varchar(40) DEFAULT NULL,
	// `password` text NOT NULL

	// Prepare query
	query := `SELECT first_name, last_name, email, phone, birth_date, gender, notification_method, subscribed_notifications, state FROM profiles WHERE email=? OR phone=?`

	// Execute query
	row := db.QueryRowContext(ctx, query, email, phoneNumber)

	profile := &account.Profile{}
	notificationMethod := ""
	subscribedNotifcations := make([]byte, 0)
	state := 0

	err := row.Scan(
		&profile.FirstName,
		&profile.LastName,
		&profile.EmailAddress,
		&profile.PhoneNumber,
		&profile.BirthDate,
		&profile.Gender,
		&notificationMethod,
		&subscribedNotifcations,
		&state,
	)
	if err != nil {
		return nil, errQueryFailed(err, "GetProfile (SELECT)")
	}

	if state != 1 {
		return nil, errAccountBlocked()
	}

	if err = json.Unmarshal(subscribedNotifcations, profile.SubscribeNotifications); err != nil {
		return nil, errFromJSONUnMarshal(err, "Profile.SubscribeNotifications")
	}

	profile.NotificationMethod = account.NotificationMethod(account.NotificationMethod_value[notificationMethod])

	return profile, nil
}
