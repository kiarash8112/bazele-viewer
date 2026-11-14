// Package auth signs customers up and looks them up by token.
package auth

import (
	"errors"

	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/pkg/ids"
	"example.com/bazelviewer/example/pkg/validate"
)

// ErrUnauthorized means the token belongs to no customer.
var ErrUnauthorized = errors.New("unauthorized")

// Service manages customer accounts.
type Service struct {
	db *db.DB
}

// New returns a Service that stores accounts in d.
func New(d *db.DB) *Service { return &Service{db: d} }

// SignUp creates an account for email and returns its token.
func (s *Service) SignUp(email string) (string, error) {
	if err := validate.Email(email); err != nil {
		return "", err
	}
	token := ids.New("tok")
	s.db.Insert("users", map[string]string{"email": email, "token": token})
	return token, nil
}

// Customer returns the email of the account with token.
func (s *Service) Customer(token string) (string, error) {
	for _, r := range s.db.All("users") {
		if r.Fields["token"] == token {
			return r.Fields["email"], nil
		}
	}
	return "", ErrUnauthorized
}
