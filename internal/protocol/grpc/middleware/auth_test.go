package middleware

import (
	"context"
	"testing"

	account "github.com/gidyon/rupacinema/account/pkg/api"
)

func TestGenAndParseToken(t *testing.T) {
	type args struct {
		ctx     context.Context
		profile *account.Profile
		admin   *account.Admin
	}
	tests := []struct {
		name     string
		args     args
		genErr   bool
		parseErr bool
	}{
		{
			"Generate and Parse Valid Token",
			args{context.Background(), &account.Profile{FirstName: "max", LastName: "Masika", EmailAddress: "x"}, &account.Admin{}},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenToken(tt.args.ctx, tt.args.profile, tt.args.admin)
			if (err != nil) != tt.genErr {
				t.Errorf("GenToken() error = %v, wantErr %v", err, tt.genErr)
				return
			}
			claims, err := parseToken(got)
			if err != nil {
				t.Errorf("parseToken() error = %v, wantErr %v", err, tt.parseErr)
				return
			}
			switch {
			case claims.Profile.FirstName != tt.args.profile.FirstName:
				t.Errorf("claims Profile FirstName = %v, want %v", claims.Profile.FirstName, tt.args.profile.FirstName)
			case claims.Profile.LastName != tt.args.profile.LastName:
				t.Errorf("claims Profile LastName = %v, want %v", claims.Profile.LastName, tt.args.profile.LastName)
			case claims.Profile.EmailAddress != tt.args.profile.EmailAddress:
				t.Errorf("claims Profile EmailAddress = %v, want %v", claims.Profile.EmailAddress, tt.args.profile.EmailAddress)
			}
		})
	}
}
