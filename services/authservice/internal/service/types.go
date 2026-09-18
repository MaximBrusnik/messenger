package service

import (
	"time"

	"messengermax/authservice/internal/entity"
)

type AuthResult struct {
	Token                     string
	User                      *entity.User
	EmailVerificationRequired bool
}

type DeviceInfo struct {
	Name        string
	Platform    string
	Fingerprint string
	IP          string
}

type DeviceSummary struct {
	Name       string
	Platform   string
	IP         string
	FirstLogin time.Time
	LastLogin  time.Time
	LoginCount int
	IsCurrent  bool
}
