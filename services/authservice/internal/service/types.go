package service

import "messengermax/authservice/internal/entity"

type AuthResult struct {
	Token                     string
	User                      *entity.User
	EmailVerificationRequired bool
}
