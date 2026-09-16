package service

import "messengermax/userservice/internal/entity"

type ProfileView struct {
	Profile entity.Profile
	Online  bool
}

type ProfileUpdate struct {
	Username    string
	Email       string
	Avatar      string
	Bio         string
	DateOfBirth string
	ClearAvatar bool
	IsAdmin     *bool
	IsBot       *bool
}
