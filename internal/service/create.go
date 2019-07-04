package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/notification/pkg/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type createUserDS struct {
	res *empty.Empty
	err error
}

func (createUser *createUserDS) Create(
	ctx context.Context,
	sqlWorkerChan chan<- sqlWorker,
	createReq *account.CreateUserRequest,
	db *sql.DB,
	notificationServiceClient notification.NotificationServiceClient,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	profile := createReq.GetProfile()
	if profile == nil {
		createUser.err = errMissingCredential("Profile must be non-nil")
		return
	}

	// Validate the profile information
	err := func() error {
		var err error
		switch {
		case strings.Trim(
			profile.EmailAddress, " ",
		) == "" && strings.Trim(
			profile.PhoneNumber, " ",
		) == "":
			err = errMissingCredential("email address or phone number")
		case strings.Trim(profile.FirstName, " ") == "":
			err = errMissingCredential("first name")
		case strings.Trim(profile.LastName, " ") == "":
			err = errMissingCredential("last name")
		}
		return err
	}()
	if err != nil {
		createUser.err = err
		return
	}

	err = insertUserToDB(ctx, db, createReq)
	if err != nil {
		createUser.err = err
		return
	}

	createUser.res = &empty.Empty{}

	// Notify user
	notificationServiceClient.Trigger(
		ctx,
		&notification.Notification{
			NotificationId: uuid.New().String(),
			Priority:       notification.Priority_MEDIUM,
			SendMethod:     notification.SendMethod_EMAIL,
			CreateTime:     &timestamp.Timestamp{Nanos: int32(time.Now().Nanosecond())},
			EmailNotification: &notification.EmailNotification{
				To:              profile.EmailAddress,
				Subject:         "Rup Cinema Account",
				BodyContentType: "text/html",
				Body: fmt.Sprintf(
					"Hi %s %s, Account created successful. You are now a member of rupacinema",
					profile.FirstName, profile.LastName,
				),
			},
			SmsNotification: &notification.SMSNotification{
				Message: fmt.Sprintf(
					"Hi %s, Account created successful. You are now a member of rupacinema",
					profile.FirstName+profile.LastName,
				),
			},
			Bulk: false,
			Save: false,
		},
	)
}

func insertUserToDB(ctx context.Context, db *sql.DB, createReq *account.CreateUserRequest) error {
	// Check if context is cancelled
	if cancelled(ctx) {
		return ctx.Err()
	}

	profile := createReq.GetProfile()
	privateProfile := createReq.GetPrivateProfile()

	if strings.Trim(privateProfile.Password, " ") != "" {
		newPass, err := generateHashPassword(privateProfile.Password)
		if err != nil {
			return errFailedToGenHashedPass(err)
		}
		privateProfile.Password = newPass
	}

	firstName := ""
	// Check the user does not exist
	query := `SELECT first_name FROM users WHERE email=? OR phone=?`
	// Execute query and populate result
	row := db.QueryRow(query, profile.EmailAddress, profile.PhoneNumber)
	err := row.Scan(&firstName)
	switch err {
	case nil:
		return errAccountDoesExist()
	case sql.ErrNoRows:
	default:
		return errQueryFailed(err, "CreateUser (SELECT)")
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
	query = `INSERT INTO users (first_name, last_name, email, phone, birth_date, gender, state, security_question, security_answer, password) VALUES(?, ?, ?, ?, DATE(?), ?, ?, ?, ?, ?)`

	// Execute query
	_, err = db.ExecContext(ctx, query,
		profile.FirstName,
		profile.LastName,
		profile.EmailAddress,
		profile.PhoneNumber,
		profile.BirthDate,
		profile.Gender,
		1,
		privateProfile.SecurityQuestion,
		privateProfile.SecurityAnswer,
		privateProfile.Password,
	)
	if err != nil {
		return errQueryFailed(err, "CreateProfile (INSERT)")
	}

	return nil
}

func generateHashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func compareHashedPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

type createAdminDS struct {
	res *empty.Empty
	err error
}

func (createAdmin *createAdminDS) Create(
	ctx context.Context,
	sqlWorkerChan chan<- sqlWorker,
	createReq *account.CreateAdminRequest,
	db *sql.DB,
	notificationServiceClient notification.NotificationServiceClient,
) {
	// Check if context is cancelled before proceeding
	if cancelled(ctx) {
		return
	}

	admin := createReq.GetNewAdmin()
	if admin == nil {
		createAdmin.err = errMissingCredential("Admin data must be non-nil")
		return
	}

	// Validate the profile information
	err := func() error {
		var err error
		switch {
		case strings.Trim(admin.EmailAddress, " ") == "":
			err = errMissingCredential("email address")
		case strings.Trim(admin.FirstName, " ") == "":
			err = errMissingCredential("first name")
		case strings.Trim(admin.LastName, " ") == "":
			err = errMissingCredential("last name")
		}
		return err
	}()
	if err != nil {
		createAdmin.err = err
		return
	}

	err = insertAdminToDB(ctx, db, createReq)
	if err != nil {
		createAdmin.err = err
		return
	}

	createAdmin.res = &empty.Empty{}

	// Notify admin
	notificationServiceClient.Trigger(
		ctx,
		&notification.Notification{
			NotificationId: uuid.New().String(),
			Priority:       notification.Priority_MEDIUM,
			SendMethod:     notification.SendMethod_EMAIL,
			CreateTime:     &timestamp.Timestamp{Nanos: int32(time.Now().Nanosecond())},
			EmailNotification: &notification.EmailNotification{
				From:            "Rupa Cinema Group",
				To:              admin.EmailAddress,
				Subject:         "Rup Cinema Group",
				BodyContentType: "text/html",
				Body: fmt.Sprintf(
					"Hi %s %s, Account created successful\n. You are now an admin at the rupacinema.",
					admin.FirstName, admin.LastName,
				),
			},
			SmsNotification: &notification.SMSNotification{
				Message: fmt.Sprintf(
					"Hi %s %s, Account created successful. You are now an admin at the rupacinema.",
					admin.FirstName, admin.LastName,
				),
			},
			Bulk: false,
			Save: false,
		},
	)

}

func insertAdminToDB(
	ctx context.Context, db *sql.DB, createAdmin *account.CreateAdminRequest,
) error {
	// Check if context is cancelled
	if cancelled(ctx) {
		return ctx.Err()
	}

	admin := createAdmin.GetNewAdmin()
	privateProfile := createAdmin.GetAdminPrivate()

	newPass, err := generateHashPassword(privateProfile.Password)
	if err != nil {
		return errFailedToGenHashedPass(err)
	}
	privateProfile.Password = newPass

	trustedDevices, err := json.Marshal(admin.TrustedDevices)
	if err != nil {
		return errFromJSONMarshal(err, "Admin.TrustedDevices")
	}
	if len(trustedDevices) == 0 {
		trustedDevices = []byte("[]")
	}

	// `first_name` varchar(50) NOT NULL,
	// `last_name` varchar(50) NOT NULL,
	// `email` varchar(50) NOT NULL,
	// `phone` varchar(15) NOT NULL,
	// `user_name` varchar(50) NOT NULL,
	// `admin_level` enum('READER','READER_AND_WRITE_ONLY_FOOD','SUPER_ADMIN') NOT NULL DEFAULT 'READER',
	// `trusted_devices` json NOT NULL,
	// `password` text NOT NULL

	// Prepare query
	query := `INSERT INTO admins (first_name, last_name, email, phone, user_name, admin_level, trusted_devices, password) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`

	// Execute query
	_, err = db.ExecContext(ctx, query,
		admin.FirstName,
		admin.LastName,
		admin.EmailAddress,
		admin.PhoneNumber,
		admin.UserName,
		account.AdminLevel_name[int32(admin.Level)],
		trustedDevices,
		privateProfile.Password,
	)
	if err != nil {
		return errQueryFailed(err, "CreateAdmin (INSERT)")
	}

	return nil
}
