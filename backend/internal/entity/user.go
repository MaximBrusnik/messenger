package entity

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username          string     `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Email             string     `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password          string     `gorm:"size:255;not null" json:"-"`
	LastLogin         time.Time  `json:"last_login,omitempty"`
	IsActive          bool       `gorm:"default:true" json:"is_active"`
	Avatar            string     `gorm:"size:500" json:"avatar,omitempty"`
	Status            string     `gorm:"size:50;default:'offline'" json:"status"`
	ShowOnlineStatus  bool       `gorm:"default:true" json:"show_online_status"`
	LastSeenPrivacy   string     `gorm:"size:20;default:'everyone'" json:"last_seen_privacy"`
	AvatarPrivacy     string     `gorm:"size:20;default:'everyone'" json:"avatar_privacy"`
	EmailVerified     bool       `gorm:"default:false" json:"email_verified"`
	VerificationToken string     `gorm:"size:255" json:"-"`
	Bio               string     `gorm:"size:500" json:"bio,omitempty"`
	DateOfBirth       *time.Time `json:"date_of_birth,omitempty"`
	IsBot             bool       `gorm:"default:false" json:"is_bot"`

	Chats    []Chat    `gorm:"many2many:chat_users;" json:"-"`
	Messages []Message `gorm:"foreignKey:SenderID" json:"-"`
	Contacts []User    `gorm:"many2many:user_contacts;joinForeignKey:user_id;joinReferences:contact_id" json:"-"`
}

type UpdateProfileRequest struct {
	Username    string  `json:"username" binding:"omitempty,min=3,max=100"`
	Email       string  `json:"email" binding:"omitempty,email"`
	Avatar      *string `json:"avatar,omitempty"`
	Bio         string  `json:"bio,omitempty"`
	DateOfBirth string  `json:"date_of_birth,omitempty"`
}

type UpdateSettingsRequest struct {
	ShowOnlineStatus *bool  `json:"show_online_status,omitempty"`
	LastSeenPrivacy  string `json:"last_seen_privacy,omitempty" binding:"omitempty,oneof=everyone contacts nobody"`
	AvatarPrivacy    string `json:"avatar_privacy,omitempty" binding:"omitempty,oneof=everyone contacts nobody"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type SearchUsersRequest struct {
	Query string `form:"q" binding:"required,min=1"`
}

type AddContactRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// WebSocket сообщения
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// DTO для запросов
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID            uint      `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastLogin     time.Time `json:"last_login,omitempty"`
	Avatar        string    `json:"avatar,omitempty"`
	Status        string    `json:"status,omitempty"`
	EmailVerified bool      `json:"email_verified"`
	Bio           string    `json:"bio,omitempty"`
	DateOfBirth   string    `json:"date_of_birth,omitempty"`
	IsBot         bool      `json:"is_bot"`
}

type UserProfileResponse struct {
	UserResponse
	CommonChats int `json:"common_chats"`
}

type UserSettingsResponse struct {
	ShowOnlineStatus bool   `json:"show_online_status"`
	LastSeenPrivacy  string `json:"last_seen_privacy"`
	AvatarPrivacy    string `json:"avatar_privacy"`
}

// Хэширование пароля
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

// Проверка пароля
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

// Преобразование в DTO
func (u *User) ToResponse() UserResponse {
	dob := ""
	if u.DateOfBirth != nil {
		dob = u.DateOfBirth.Format("2006-01-02")
	}
	return UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		CreatedAt:     u.CreatedAt,
		LastLogin:     u.LastLogin,
		Avatar:        u.Avatar,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		Bio:           u.Bio,
		DateOfBirth:   dob,
		IsBot:         u.IsBot,
	}
}

func CanSeeLastSeen(viewer *User, target *User) bool {
	switch target.LastSeenPrivacy {
	case "everyone":
		return true
	case "contacts":
		return target.ShowOnlineStatus
	case "nobody":
		return false
	default:
		return true
	}
}
