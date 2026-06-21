package logic

import (
	entity2 "MessangerMax/internal/entity"
	repo2 "MessangerMax/internal/repo"
	"errors"
	"time"
)

type ReactionService interface {
	AddReaction(userID uint, messageID uint, req entity2.AddReactionRequest) (*entity2.ReactionResponse, error)
	RemoveReaction(userID uint, messageID uint, reaction string) error
	GetMessageReactions(messageID uint) ([]entity2.ReactionResponse, error)
}

type reactionService struct {
	reactionRepo repo2.ReactionRepository
	messageRepo  repo2.MessageRepository
	notifier     Notifier
}

func NewReactionService(
	reactionRepo repo2.ReactionRepository,
	messageRepo repo2.MessageRepository,
	notifier Notifier,
) ReactionService {
	return &reactionService{
		reactionRepo: reactionRepo,
		messageRepo:  messageRepo,
		notifier:     notifier,
	}
}

func (s *reactionService) AddReaction(userID uint, messageID uint, req entity2.AddReactionRequest) (*entity2.ReactionResponse, error) {
	message, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return nil, errors.New("сообщение не найдено")
	}

	reaction := &entity2.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Reaction:  req.Reaction,
		CreatedAt: time.Now(),
	}

	if err := s.reactionRepo.Add(reaction); err != nil {
		return nil, err
	}

	response := &entity2.ReactionResponse{
		MessageID: messageID,
		UserID:    userID,
		Reaction:  req.Reaction,
		CreatedAt: reaction.CreatedAt,
	}

	s.notifier.SendReactionAdded(message.ChatID, messageID, response)
	return response, nil
}

func (s *reactionService) RemoveReaction(userID uint, messageID uint, reaction string) error {
	message, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return errors.New("сообщение не найдено")
	}

	if err := s.reactionRepo.Remove(messageID, userID, reaction); err != nil {
		return err
	}

	s.notifier.SendReactionRemoved(message.ChatID, messageID, userID, reaction)
	return nil
}

func (s *reactionService) GetMessageReactions(messageID uint) ([]entity2.ReactionResponse, error) {
	reactions, err := s.reactionRepo.FindByMessageID(messageID)
	if err != nil {
		return nil, err
	}

	var responses []entity2.ReactionResponse
	for _, r := range reactions {
		resp := entity2.ReactionResponse{
			MessageID: r.MessageID,
			UserID:    r.UserID,
			Reaction:  r.Reaction,
			CreatedAt: r.CreatedAt,
		}
		if r.User.ID != 0 {
			resp.Username = r.User.Username
		}
		responses = append(responses, resp)
	}
	return responses, nil
}
