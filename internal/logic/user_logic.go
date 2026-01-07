package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"errors"
	"strings"
)

type UserService interface {
	GetAllUsers(excludeID uint) ([]entity.UserResponse, error)
	SearchUsers(query string, excludeID uint) ([]entity.UserResponse, error)
	AddContact(userID, contactID uint) error
	RemoveContact(userID, contactID uint) error
	GetContacts(userID uint) ([]entity.UserResponse, error)
	UpdateProfile(userID uint, req entity.UpdateProfileRequest) (*entity.UserResponse, error)
	ChangePassword(userID uint, req entity.ChangePasswordRequest) error
}

type userService struct {
	userRepo repo.UserRepository
}

func NewUserService(userRepo repo.UserRepository) UserService {
	return &userService{userRepo: userRepo}
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
	// Проверяем, существует ли пользователь
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

	// Проверяем уникальность username если он меняется
	if req.Username != "" && req.Username != user.Username {
		existingUser, _ := s.userRepo.FindByUsername(req.Username)
		if existingUser != nil {
			return nil, errors.New("пользователь с таким именем уже существует")
		}
		user.Username = req.Username
	}

	// Проверяем уникальность email если он меняется
	if req.Email != "" && req.Email != user.Email {
		existingUser, _ := s.userRepo.FindByEmail(req.Email)
		if existingUser != nil {
			return nil, errors.New("пользователь с таким email уже существует")
		}
		user.Email = req.Email
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

	// Проверяем старый пароль
	if err := user.CheckPassword(req.OldPassword); err != nil {
		return errors.New("неверный старый пароль")
	}

	// Хэшируем новый пароль
	if err := user.HashPassword(req.NewPassword); err != nil {
		return errors.New("ошибка при обработке пароля")
	}

	return s.userRepo.UpdatePassword(userID, user.Password)
}
