package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"errors"
	"strings"
	"time"
)

type UserService interface {
	GetAllUsers(excludeID uint) ([]entity.UserResponse, error)
	SearchUsers(query string, excludeID uint) ([]entity.UserResponse, error)
	AddContact(userID, contactID uint) error
	RemoveContact(userID, contactID uint) error
	GetContacts(userID uint) ([]entity.UserResponse, error)
	UpdateProfile(userID uint, req entity.UpdateProfileRequest) (*entity.UserResponse, error)
	ChangePassword(userID uint, req entity.ChangePasswordRequest) error
	GetSettings(userID uint) (*entity.UserSettingsResponse, error)
	UpdateSettings(userID uint, req entity.UpdateSettingsRequest) error
	GetUserProfile(targetID, requesterID uint) (*entity.UserProfileResponse, error)
}

type userService struct {
	userRepo repo.UserRepository
	chatRepo repo.ChatRepository
}

func NewUserService(userRepo repo.UserRepository, chatRepo repo.ChatRepository) UserService {
	return &userService{userRepo: userRepo, chatRepo: chatRepo}
}

func (s *userService) GetAllUsers(excludeID uint) ([]entity.UserResponse, error) {
	users, err := s.userRepo.FindAll(excludeID)
	if err != nil {
		return nil, err
	}

	var responses []entity.UserResponse
	for _, user := range users {
		responses = append(responses, user.ToResponse())
	}

	return responses, nil
}

func (s *userService) SearchUsers(query string, excludeID uint) ([]entity.UserResponse, error) {
	users, err := s.userRepo.Search(strings.TrimSpace(query), excludeID)
	if err != nil {
		return nil, err
	}

	var responses []entity.UserResponse
	for _, user := range users {
		responses = append(responses, user.ToResponse())
	}

	return responses, nil
}

func (s *userService) AddContact(userID, contactID uint) error {
	contact, err := s.userRepo.FindByID(contactID)
	if err != nil || contact == nil {
		return errors.New("пользователь не найден")
	}

	return s.userRepo.AddContact(userID, contactID)
}

func (s *userService) RemoveContact(userID, contactID uint) error {
	return s.userRepo.RemoveContact(userID, contactID)
}

func (s *userService) GetContacts(userID uint) ([]entity.UserResponse, error) {
	contacts, err := s.userRepo.GetContacts(userID)
	if err != nil {
		return nil, err
	}

	var responses []entity.UserResponse
	for _, contact := range contacts {
		responses = append(responses, contact.ToResponse())
	}

	return responses, nil
}

func (s *userService) UpdateProfile(userID uint, req entity.UpdateProfileRequest) (*entity.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	if req.Username != "" && req.Username != user.Username {
		existingUser, _ := s.userRepo.FindByUsername(req.Username)
		if existingUser != nil {
			return nil, errors.New("пользователь с таким именем уже существует")
		}
		user.Username = req.Username
	}

	if req.Email != "" && req.Email != user.Email {
		existingUser, _ := s.userRepo.FindByEmail(req.Email)
		if existingUser != nil {
			return nil, errors.New("пользователь с таким email уже существует")
		}
		user.Email = req.Email
	}

	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}

	if req.Bio != "" || (req.Bio == "" && user.Bio != "") {
		user.Bio = req.Bio
	}

	if req.DateOfBirth != "" {
		parsed, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			user.DateOfBirth = &parsed
		}
	} else if req.DateOfBirth == "" {
		user.DateOfBirth = nil
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

func (s *userService) ChangePassword(userID uint, req entity.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("пользователь не найден")
	}

	if err := user.CheckPassword(req.OldPassword); err != nil {
		return errors.New("неверный старый пароль")
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		return errors.New("ошибка при обработке пароля")
	}

	return s.userRepo.UpdatePassword(userID, user.Password)
}

func (s *userService) GetSettings(userID uint) (*entity.UserSettingsResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}
	return &entity.UserSettingsResponse{
		ShowOnlineStatus: user.ShowOnlineStatus,
		LastSeenPrivacy:  user.LastSeenPrivacy,
		AvatarPrivacy:    user.AvatarPrivacy,
	}, nil
}

func (s *userService) UpdateSettings(userID uint, req entity.UpdateSettingsRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("пользователь не найден")
	}

	if req.ShowOnlineStatus != nil {
		user.ShowOnlineStatus = *req.ShowOnlineStatus
	}
	if req.LastSeenPrivacy != "" {
		user.LastSeenPrivacy = req.LastSeenPrivacy
	}
	if req.AvatarPrivacy != "" {
		user.AvatarPrivacy = req.AvatarPrivacy
	}

	return s.userRepo.Update(user)
}

func (s *userService) GetUserProfile(targetID, requesterID uint) (*entity.UserProfileResponse, error) {
	target, err := s.userRepo.FindByID(targetID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	isSelf := targetID == requesterID
	resp := target.ToResponse()

	if !isSelf {
		resp.Email = ""
		if !target.ShowOnlineStatus {
			resp.Status = ""
		}
		if target.LastSeenPrivacy != "everyone" {
			resp.LastLogin = time.Time{}
		}
		if target.AvatarPrivacy != "everyone" {
			if target.AvatarPrivacy == "nobody" {
				resp.Avatar = ""
			} else if target.AvatarPrivacy == "contacts" {
				isContact, _ := s.userRepo.IsContact(requesterID, targetID)
				if !isContact {
					resp.Avatar = ""
				}
			}
		}
		// Always show bio and date_of_birth if filled
	}

	commonIDs, _ := s.chatRepo.GetCommonChatIDs(requesterID, targetID)

	return &entity.UserProfileResponse{
		UserResponse: resp,
		CommonChats:  len(commonIDs),
	}, nil
}
