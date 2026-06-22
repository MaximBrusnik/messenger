package logic

import (
	entity2 "MessangerMax/internal/entity"
	repo2 "MessangerMax/internal/repo"
	"errors"
	"time"
)

type ChatService interface {
	CreateChat(userID uint, req entity2.CreateChatRequest) (*entity2.ChatResponse, error)
	GetUserChats(userID uint) ([]entity2.ChatResponse, error)
	GetChatByID(chatID, userID uint) (*entity2.ChatResponse, error)
	SendMessage(userID, chatID uint, req entity2.SendMessageRequest) (*entity2.MessageResponse, error)
	EditMessage(userID, messageID uint, req entity2.EditMessageRequest) (*entity2.MessageResponse, error)
	GetChatMessages(chatID, userID uint, limit, offset int) ([]entity2.MessageResponse, error)
	MarkAsRead(chatID, userID uint) error
	AddUserToChat(chatID, userID, targetUserID uint) error
	RemoveUserFromChat(chatID, userID, targetUserID uint) error
	DeleteChat(chatID, userID uint) error
	DeleteMessage(messageID, userID uint) error
}

type chatService struct {
	chatRepo    repo2.ChatRepository
	messageRepo repo2.MessageRepository
	userRepo    repo2.UserRepository
	notifier    Notifier
}

func NewChatService(
	chatRepo repo2.ChatRepository,
	messageRepo repo2.MessageRepository,
	userRepo repo2.UserRepository,
	notifier Notifier,
) ChatService {
	return &chatService{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		userRepo:    userRepo,
		notifier:    notifier,
	}
}

func (s *chatService) CreateChat(userID uint, req entity2.CreateChatRequest) (*entity2.ChatResponse, error) {
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
	chat := &entity2.Chat{
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
	chat.Participants = []entity2.User{*currentUser, *targetUser}

	if err := s.chatRepo.Create(chat); err != nil {
		return nil, err
	}

	return s.convertToChatResponse(chat, userID)
}

func (s *chatService) GetUserChats(userID uint) ([]entity2.ChatResponse, error) {
	chats, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	var responses []entity2.ChatResponse
	for _, chat := range chats {
		response, err := s.convertToChatResponse(&chat, userID)
		if err != nil {
			continue
		}
		responses = append(responses, *response)
	}

	return responses, nil
}

func (s *chatService) GetChatByID(chatID, userID uint) (*entity2.ChatResponse, error) {
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

func (s *chatService) SendMessage(userID, chatID uint, req entity2.SendMessageRequest) (*entity2.MessageResponse, error) {
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
	message := &entity2.Message{
		ChatID:         chatID,
		SenderID:       userID,
		Text:           req.Content,
		AttachmentType: req.AttachmentType,
		AttachmentURL:  req.AttachmentURL,
		AttachmentName: req.AttachmentName,
		AttachmentSize: req.AttachmentSize,
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// Загружаем отправителя для ответа
	sender, _ := s.userRepo.FindByID(userID)
	message.Sender = *sender

	// Уведомляем участников чата через WebSocket
	response := s.convertToMessageResponse(message, userID)
	s.notifier.SendNewMessage(chatID, response)

	return response, nil
}

func (s *chatService) EditMessage(userID, messageID uint, req entity2.EditMessageRequest) (*entity2.MessageResponse, error) {
	message, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return nil, errors.New("сообщение не найдено")
	}

	if message.SenderID != userID {
		return nil, errors.New("нельзя редактировать чужое сообщение")
	}

	if err := s.messageRepo.UpdateText(messageID, req.Content); err != nil {
		return nil, err
	}

	message.Text = req.Content
	message.Edited = true
	now := time.Now()
	message.EditedAt = &now

	response := s.convertToMessageResponse(message, userID)
	s.notifier.SendMessageEdited(message.ChatID, response)

	return response, nil
}

func (s *chatService) GetChatMessages(chatID, userID uint, limit, offset int) ([]entity2.MessageResponse, error) {
	// Проверяем доступ
	_, err := s.GetChatByID(chatID, userID)
	if err != nil {
		return nil, err
	}

	messages, err := s.messageRepo.FindByChatID(chatID, limit, offset)
	if err != nil {
		return nil, err
	}

	var responses []entity2.MessageResponse
	for _, message := range messages {
		responses = append(responses, *s.convertToMessageResponse(&message, userID))
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

	// Помечаем сообщения как прочитанные и получаем их ID
	msgIDs, err := s.messageRepo.MarkChatAsRead(chatID, userID)
	if err != nil {
		return err
	}

	// Уведомляем участников о прочтении (если есть что отмечать)
	if len(msgIDs) > 0 {
		s.notifier.SendMessagesRead(chatID, msgIDs, userID)
	}

	return nil
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

func (s *chatService) DeleteChat(chatID, userID uint) error {
	chat, err := s.chatRepo.FindByID(chatID)
	if err != nil {
		return errors.New("чат не найден")
	}

	isParticipant := false
	for _, p := range chat.Participants {
		if p.ID == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return errors.New("доступ запрещен")
	}

	if err := s.chatRepo.Delete(chatID); err != nil {
		return err
	}

	s.notifier.SendChatDeleted(chatID)
	return nil
}

func (s *chatService) DeleteMessage(messageID, userID uint) error {
	message, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return errors.New("сообщение не найдено")
	}

	if message.SenderID != userID {
		return errors.New("нельзя удалить чужое сообщение")
	}

	if err := s.messageRepo.Delete(messageID); err != nil {
		return err
	}

	s.notifier.SendMessageDeleted(message.ChatID, messageID)
	return nil
}

// Вспомогательные методы
func (s *chatService) convertToChatResponse(chat *entity2.Chat, currentUserID uint) (*entity2.ChatResponse, error) {
	// Получаем последнее сообщение
	lastMessage, _ := s.messageRepo.GetLastMessage(chat.ID)

	// Получаем количество непрочитанных
	unreadCount, _ := s.chatRepo.GetUnreadCount(chat.ID, currentUserID)

	// Создаем ответ
	response := &entity2.ChatResponse{
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
			avatar := participant.Avatar
			if participant.AvatarPrivacy == "nobody" {
				avatar = ""
			} else if participant.AvatarPrivacy == "contacts" {
				isContact, _ := s.userRepo.IsContact(currentUserID, participant.ID)
				if !isContact {
					avatar = ""
				}
			}

			response.Participants = append(response.Participants, entity2.UserResponse{
				ID:        participant.ID,
				Username:  participant.Username,
				Email:     participant.Email,
				Avatar:    avatar,
				Status:    participant.Status,
				LastLogin: participant.LastLogin,
				CreatedAt: participant.CreatedAt,
			})
		}
	}

	// Добавляем последнее сообщение
	if lastMessage != nil {
		response.LastMessage = &entity2.MessageResponse{
			ID:        lastMessage.ID,
			ChatID:    lastMessage.ChatID,
			SenderID:  lastMessage.SenderID,
			Text:      lastMessage.Text,
			IsRead:    lastMessage.IsRead,
			ReadAt:    lastMessage.ReadAt,
			CreatedAt: lastMessage.CreatedAt,
		}
	}

	return response, nil
}

func (s *chatService) convertToMessageResponse(message *entity2.Message, viewerID uint) *entity2.MessageResponse {
	response := &entity2.MessageResponse{
		ID:             message.ID,
		ChatID:         message.ChatID,
		SenderID:       message.SenderID,
		Text:           message.Text,
		IsRead:         message.IsRead,
		ReadAt:         message.ReadAt,
		CreatedAt:      message.CreatedAt,
		Edited:         message.Edited,
		EditedAt:       message.EditedAt,
		AttachmentType: message.AttachmentType,
		AttachmentURL:  message.AttachmentURL,
		AttachmentName: message.AttachmentName,
		AttachmentSize: message.AttachmentSize,
	}

	if message.Sender.ID != 0 {
		avatar := message.Sender.Avatar
		if viewerID != message.Sender.ID {
			if message.Sender.AvatarPrivacy == "nobody" {
				avatar = ""
			} else if message.Sender.AvatarPrivacy == "contacts" {
				isContact, _ := s.userRepo.IsContact(viewerID, message.Sender.ID)
				if !isContact {
					avatar = ""
				}
			}
		}

		response.Sender = &entity2.UserResponse{
			ID:        message.Sender.ID,
			Username:  message.Sender.Username,
			Email:     message.Sender.Email,
			Avatar:    avatar,
			Status:    message.Sender.Status,
			LastLogin: message.Sender.LastLogin,
			CreatedAt: message.Sender.CreatedAt,
		}
	}

	if len(message.Reactions) > 0 {
		for _, r := range message.Reactions {
			rr := entity2.ReactionResponse{
				MessageID: r.MessageID,
				UserID:    r.UserID,
				Reaction:  r.Reaction,
				CreatedAt: r.CreatedAt,
			}
			if r.User.ID != 0 {
				rr.Username = r.User.Username
			}
			response.Reactions = append(response.Reactions, rr)
		}
	}

	return response
}
