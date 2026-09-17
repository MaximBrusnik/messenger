package entity

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Username          string         `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Email             string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password          string         `gorm:"size:255;not null" json:"-"`
	LastLogin         time.Time      `json:"last_login,omitempty"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	EmailVerified     bool           `gorm:"default:false" json:"email_verified"`
	VerificationToken string         `gorm:"size:255" json:"-"`
	IsBot             bool           `gorm:"default:false" json:"is_bot"`
	IsAdmin           bool           `gorm:"default:false" json:"is_admin"`
	Avatar            string         `gorm:"size:500" json:"avatar,omitempty"`
	Status            string         `gorm:"size:50;default:'offline'" json:"status"`
}

func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
