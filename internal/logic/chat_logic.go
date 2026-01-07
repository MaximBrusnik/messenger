package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"errors"
)

type ChatService interface {
	CreateChat(userID uint, req entity.CreateChatRequest) (*entity.ChatResponse, error)
	GetUserChats(userID uint) ([]entity.ChatResponse, error)
	GetChatByID(chatID, userID uint) (*entity.ChatResponse, error)
	SendMessage(userID, chatID uint, req entity.SendMessageRequest) (*entity.MessageResponse, error)
	GetChatMessages(chatID, userID uint, limit, offset int) ([]entity.MessageResponse, error)
	MarkAsRead(chatID, userID uint) error
	AddUserToChat(chatID, userID, targetUserID uint) error
	RemoveUserFromChat(chatID, userID, targetUserID uint) error
}

type chatService struct {
	chatRepo    repo.ChatRepository
	messageRepo repo.MessageRepository
	userRepo    repo.UserRepository
}

func NewChatService(
	chatRepo repo.ChatRepository,
	messageRepo repo.MessageRepository,
	userRepo repo.UserRepository,
) ChatService {
	return &chatService{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

func (s *chatService) CreateChat(userID uint, req entity.CreateChatRequest) (*entity.ChatResponse, error) {
	// Проверяем, существует ли пользователь
	targetUser, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	// Для приватного чата проверяем, не существует ли уже такой чат
	if req.Type == "" || req.Type == "private" {
		existingChat, err := s.chatRepo.FindPrivateChat(userID, req.UserID)
		if err == nil && existingChat != nil {
			return s.convertToChatResponse(existingChat, userID)
		}
	}

	// Создаем чат
	chat := &entity.Chat{
		Type: req.Type,
		Name: req.Name,
	}

	if chat.Type == "" {
		chat.Type = "private"
	}

	if chat.Type == "private" && chat.Name == "" {
		chat.Name = targetUser.Username
	}

	// Добавляем участников
	currentUser, _ := s.userRepo.FindByID(userID)
	chat.Participants = []entity.User{*currentUser, *targetUser}

	if err := s.chatRepo.Create(chat); err != nil {
		return nil, err
	}

	return s.convertToChatResponse(chat, userID)
}

func (s *chatService) GetUserChats(userID uint) ([]entity.ChatResponse, error) {
	chats, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	var responses []entity.ChatResponse
	for _, chat := range chats {
		response, err := s.convertToChatResponse(&chat, userID)
		if err != nil {
			continue
		}
		responses = append(responses, *response)
	}

	return responses, nil
}

func (s *chatService) GetChatByID(chatID, userID uint) (*entity.ChatResponse, error) {
	chat, err := s.chatRepo.FindByID(chatID)
	if err != nil {
		return nil, errors.New("чат не найден")
	}

	// Проверяем, является ли пользователь участником чата
	isParticipant := false
	for _, participant := range chat.Participants {
		if participant.ID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return nil, errors.New("доступ запрещен")
	}

	return s.convertToChatResponse(chat, userID)
}

func (s *chatService) SendMessage(userID, chatID uint, req entity.SendMessageRequest) (*entity.MessageResponse, error) {
	// Проверяем, существует ли чат и является ли пользователь участником
	chat, err := s.chatRepo.FindByID(chatID)
	if err != nil {
		return nil, errors.New("чат не найден")
	}

	isParticipant := false
	for _, participant := range chat.Participants {
		if participant.ID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return nil, errors.New("доступ запрещен")
	}

	// Создаем сообщение
	message := &entity.Message{
		ChatID:   chatID,
		SenderID: userID,
		Text:     req.Text,
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// Загружаем отправителя для ответа
	sender, _ := s.userRepo.FindByID(userID)
	message.Sender = *sender

	// Обновляем время обновления чата
	//s.chatRepo.db.Model(chat).Update("updated_at", message.CreatedAt)

	return s.convertToMessageResponse(message), nil
}

func (s *chatService) GetChatMessages(chatID, userID uint, limit, offset int) ([]entity.MessageResponse, error) {
	// Проверяем доступ
	_, err := s.GetChatByID(chatID, userID)
	if err != nil {
		return nil, err
	}

	messages, err := s.messageRepo.FindByChatID(chatID, limit, offset)
	if err != nil {
		return nil, err
	}

	var responses []entity.MessageResponse
	for _, message := range messages {
		responses = append(responses, *s.convertToMessageResponse(&message))
	}

	return responses, nil
}

func (s *chatService) MarkAsRead(chatID, userID uint) error {
	// Проверяем доступ
	_, err := s.GetChatByID(chatID, userID)
	if err != nil {
		return err
	}

	// Обновляем время последнего прочтения
	if err := s.chatRepo.UpdateLastRead(chatID, userID); err != nil {
		return err
	}

	// Помечаем сообщения как прочитанные
	return s.messageRepo.MarkChatAsRead(chatID, userID)
}

func (s *chatService) AddUserToChat(chatID, userID, targetUserID uint) error {
	// Проверяем права (только участники могут добавлять)
	_, err := s.GetChatByID(chatID, userID)
	if err != nil {
		return err
	}

	return s.chatRepo.AddUserToChat(chatID, targetUserID)
}

func (s *chatService) RemoveUserFromChat(chatID, userID, targetUserID uint) error {
	// Проверяем права
	_, err := s.GetChatByID(chatID, userID)
	if err != nil {
		return err
	}

	return s.chatRepo.RemoveUserFromChat(chatID, targetUserID)
}

// Вспомогательные методы
func (s *chatService) convertToChatResponse(chat *entity.Chat, currentUserID uint) (*entity.ChatResponse, error) {
	// Получаем последнее сообщение
	lastMessage, _ := s.messageRepo.GetLastMessage(chat.ID)

	// Получаем количество непрочитанных
	unreadCount, _ := s.chatRepo.GetUnreadCount(chat.ID, currentUserID)

	// Создаем ответ
	response := &entity.ChatResponse{
		ID:        chat.ID,
		Name:      chat.Name,
		Type:      chat.Type,
		CreatedAt: chat.CreatedAt,
		UpdatedAt: chat.UpdatedAt,
		Unread:    unreadCount,
	}

	// Добавляем участников (без текущего пользователя)
	for _, participant := range chat.Participants {
		if participant.ID != currentUserID {
			response.Participants = append(response.Participants, entity.UserResponse{
				ID:       participant.ID,
				Username: participant.Username,
				Email:    participant.Email,
			})
		}
	}

	// Добавляем последнее сообщение
	if lastMessage != nil {
		response.LastMessage = &entity.MessageResponse{
			ID:        lastMessage.ID,
			ChatID:    lastMessage.ChatID,
			SenderID:  lastMessage.SenderID,
			Text:      lastMessage.Text,
			IsRead:    lastMessage.IsRead,
			CreatedAt: lastMessage.CreatedAt,
		}
	}

	return response, nil
}

func (s *chatService) convertToMessageResponse(message *entity.Message) *entity.MessageResponse {
	response := &entity.MessageResponse{
		ID:        message.ID,
		ChatID:    message.ChatID,
		SenderID:  message.SenderID,
		Text:      message.Text,
		IsRead:    message.IsRead,
		CreatedAt: message.CreatedAt,
	}

	// Добавляем информацию об отправителе
	if message.Sender.ID != 0 {
		response.Sender = &entity.UserResponse{
			ID:       message.Sender.ID,
			Username: message.Sender.Username,
			Email:    message.Sender.Email,
		}
	}

	return response
}
