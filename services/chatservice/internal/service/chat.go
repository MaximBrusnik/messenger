package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"messengermax/chatservice/internal/entity"
	"messengermax/chatservice/internal/repo"
	"messengermax/pkg/nats"
)

const BotUsername = "Ассистент"

type Server struct {
	chatRepo     repo.ChatRepository
	messageRepo  repo.MessageRepository
	reactionRepo repo.ReactionRepository
	producer     *nats.Producer
	botID        uint
	mu           sync.Mutex
	resolveBot   func(ctx context.Context) (uint, error)
}

func NewServer(
	chatRepo repo.ChatRepository,
	messageRepo repo.MessageRepository,
	reactionRepo repo.ReactionRepository,
	producer *nats.Producer,
	botID uint,
	resolveBot func(ctx context.Context) (uint, error),
) *Server {
	return &Server{
		chatRepo:     chatRepo,
		messageRepo:  messageRepo,
		reactionRepo: reactionRepo,
		producer:     producer,
		botID:        botID,
		resolveBot:   resolveBot,
	}
}

func (s *Server) ensureBot(ctx context.Context) {
	if s.botID != 0 || s.resolveBot == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.botID != 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	id, err := s.resolveBot(ctx)
	if err != nil || id == 0 {
		log.Printf("chat: lazy AI assistant resolve failed: %v", err)
		return
	}
	s.botID = id
	log.Printf("chat: AI assistant resolved lazily, id=%d", id)
}

func (s *Server) CreateChat(ctx context.Context, creatorID uint, chatType, name string, participantIDs []uint) (*ChatResponse, error) {
	if chatType == "" {
		chatType = "private"
	}
	if chatType == "private" {
		if len(participantIDs) != 1 {
			return nil, errInvalid("личный чат требует одного участника")
		}
		if existing, err := s.chatRepo.FindPrivateChat(creatorID, participantIDs[0]); err == nil {
			return s.loadChatResponse(ctx, existing.ID, creatorID)
		}
	}

	chat := &entity.Chat{Name: name, Type: chatType}
	ids := make([]uint, 0, len(participantIDs)+1)
	ids = append(ids, creatorID)
	ids = append(ids, participantIDs...)
	if err := s.chatRepo.Create(chat, ids); err != nil {
		return nil, errInternal("не удалось создать чат")
	}
	return s.loadChatResponse(ctx, chat.ID, creatorID)
}

func (s *Server) GetUserChats(ctx context.Context, userID uint) ([]ChatResponse, error) {
	chats, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		return nil, errInternal("не удалось получить чаты")
	}
	var res []ChatResponse
	for _, c := range chats {
		ids, err := s.chatRepo.GetParticipantIDs(c.ID)
		if err != nil {
			continue
		}
		if len(ids) == 2 && contains(ids, s.botID) {
			continue
		}
		// skip user's own favorites (self) chat from the main list
		if isSelfChat(ids, userID) {
			continue
		}
		pc, err := s.loadChatResponse(ctx, c.ID, userID)
		if err != nil {
			continue
		}
		res = append(res, *pc)
	}
	return res, nil
}

func (s *Server) GetChat(ctx context.Context, chatID, viewerID uint) (*ChatResponse, error) {
	return s.loadChatResponse(ctx, chatID, viewerID)
}

func (s *Server) SendMessage(ctx context.Context, chatID, senderID uint, content, attachmentURL, attachmentType, attachmentName string, attachmentSize int64) (*entity.Message, error) {
	if err := s.requireParticipant(chatID, senderID); err != nil {
		return nil, err
	}
	if content == "" && attachmentURL == "" {
		return nil, errInvalid("сообщение пустое")
	}
	var size *int
	if attachmentSize > 0 {
		n := int(attachmentSize)
		size = &n
	}
	msg := &entity.Message{
		ChatID:         chatID,
		SenderID:       senderID,
		Text:           content,
		AttachmentType: attachmentType,
		AttachmentURL:  attachmentURL,
		AttachmentName: attachmentName,
		AttachmentSize: size,
	}
	if err := s.messageRepo.Create(msg); err != nil {
		return nil, errInternal("не удалось сохранить сообщение")
	}

	s.publishCreated(chatID, msg, nil)
	return msg, nil
}

func (s *Server) GetMessages(ctx context.Context, chatID, userID uint, limit, offset int) ([]MessageWithReactions, error) {
	if err := s.requireParticipant(chatID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	messages, err := s.messageRepo.FindByChatID(chatID, limit, offset)
	if err != nil {
		return nil, errInternal("не удалось получить сообщения")
	}
	res := make([]MessageWithReactions, 0, len(messages))
	for i := range messages {
		reactions, _ := s.reactionRepo.FindByMessageID(messages[i].ID)
		res = append(res, MessageWithReactions{Msg: &messages[i], Reactions: reactions})
	}
	return res, nil
}

func (s *Server) EditMessage(ctx context.Context, senderID, messageID uint, content string) (*entity.Message, error) {
	msg, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return nil, errNotFound("сообщение не найдено")
	}
	if msg.SenderID != senderID {
		return nil, errPermissionDenied("нельзя редактировать чужое сообщение")
	}
	if err := s.messageRepo.UpdateText(msg.ID, content); err != nil {
		return nil, errInternal("не удалось изменить сообщение")
	}
	fresh, err := s.messageRepo.FindByID(msg.ID)
	if err != nil {
		return nil, errInternal("не удалось загрузить сообщение")
	}
	s.producer.Publish(nats.TopicMessageEdited, itoa(uint64(msg.ChatID)), nats.EventMessageEdited{
		ChatID:  int64(fresh.ChatID),
		Message: toMessageDTO(fresh, nil),
	})
	return fresh, nil
}

func (s *Server) DeleteMessage(ctx context.Context, chatID, senderID, messageID uint) error {
	msg, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return errNotFound("сообщение не найдено")
	}
	if msg.ChatID != chatID {
		return errInvalid("неверный чат")
	}
	if msg.SenderID != senderID {
		return errPermissionDenied("нельзя удалить чужое сообщение")
	}
	if err := s.messageRepo.Delete(msg.ID); err != nil {
		return errInternal("не удалось удалить сообщение")
	}
	s.producer.Publish(nats.TopicMessageDeleted, itoa(uint64(msg.ChatID)), nats.EventMessageDeleted{
		ChatID: int64(msg.ChatID), MessageID: int64(msg.ID),
	})
	return nil
}

func (s *Server) MarkAsRead(ctx context.Context, chatID, userID uint) error {
	if err := s.requireParticipant(chatID, userID); err != nil {
		return err
	}
	ids, err := s.messageRepo.MarkChatAsRead(chatID, userID)
	if err != nil {
		return errInternal("не удалось отметить прочитанным")
	}
	if len(ids) > 0 {
		_ = s.chatRepo.UpdateLastRead(chatID, userID)
		idss := make([]int64, 0, len(ids))
		for _, id := range ids {
			idss = append(idss, int64(id))
		}
		s.producer.Publish(nats.TopicReadReceived, itoa(uint64(chatID)), nats.EventMessagesRead{
			ChatID: int64(chatID), UserID: int64(userID), MessageIDs: idss,
		})
	}
	return nil
}

func (s *Server) PinMessage(ctx context.Context, chatID, userID, messageID uint) (*entity.Message, []entity.MessageReaction, error) {
	msg, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return nil, nil, errNotFound("сообщение не найдено")
	}
	if msg.ChatID != chatID {
		return nil, nil, errInvalid("неверный чат")
	}
	if err := s.chatRepo.PinMessage(chatID, msg.ID); err != nil {
		return nil, nil, errInternal("не удалось закрепить")
	}
	system := &entity.Message{ChatID: msg.ChatID, SenderID: userID, Text: "", SystemType: "pin"}
	_ = s.messageRepo.Create(system)
	reactions, _ := s.reactionRepo.FindByMessageID(msg.ID)
	s.producer.Publish(nats.TopicMessagePinned, itoa(uint64(chatID)), nats.EventMessagePinned{
		ChatID:  int64(chatID),
		Message: toMessageDTO(msg, reactions),
	})
	return msg, reactions, nil
}

func (s *Server) UnpinMessage(ctx context.Context, chatID, userID uint) error {
	if err := s.chatRepo.UnpinMessage(chatID); err != nil {
		return errInternal("не удалось открепить")
	}
	system := &entity.Message{ChatID: chatID, SenderID: userID, Text: "", SystemType: "unpin"}
	_ = s.messageRepo.Create(system)
	s.producer.Publish(nats.TopicMessageUnpinned, itoa(uint64(chatID)), nats.EventMessagePinned{
		ChatID: int64(chatID),
	})
	return nil
}

func (s *Server) DeleteChat(ctx context.Context, chatID, userID uint) error {
	if err := s.requireParticipant(chatID, userID); err != nil {
		return err
	}
	ids, _ := s.chatRepo.GetParticipantIDs(chatID)
	if err := s.chatRepo.Delete(chatID); err != nil {
		return errInternal("не удалось удалить чат")
	}
	users := make([]int64, 0, len(ids))
	for _, id := range ids {
		users = append(users, int64(id))
	}
	s.producer.Publish(nats.TopicChatDeleted, itoa(uint64(chatID)), nats.EventChatDeleted{
		ChatID: int64(chatID), Users: users,
	})
	return nil
}

func (s *Server) GetOrCreateAIChat(ctx context.Context, userID uint) (*ChatResponse, error) {
	s.ensureBot(ctx)
	if s.botID == 0 {
		return nil, errUnavailable("AI не настроен")
	}
	if existing, err := s.chatRepo.FindPrivateChat(userID, s.botID); err == nil {
		return s.loadChatResponse(ctx, existing.ID, userID)
	}
	chat := &entity.Chat{Type: "private"}
	if err := s.chatRepo.Create(chat, []uint{userID, s.botID}); err != nil {
		return nil, errInternal("не удалось создать AI-чат")
	}
	return s.loadChatResponse(ctx, chat.ID, userID)
}

func (s *Server) GetOrCreateFavoritesChat(ctx context.Context, userID uint) (*ChatResponse, error) {
	if existing, err := s.chatRepo.FindFavoritesChat(userID); err == nil {
		return s.loadChatResponse(ctx, existing.ID, userID)
	}
	chat := &entity.Chat{Type: "private"}
	if err := s.chatRepo.Create(chat, []uint{userID}); err != nil {
		return nil, errInternal("не удалось создать чат «Избранное»")
	}
	return s.loadChatResponse(ctx, chat.ID, userID)
}

func (s *Server) ArchiveChat(ctx context.Context, chatID, userID uint) error {
	if err := s.requireParticipant(chatID, userID); err != nil {
		return err
	}
	ids, _ := s.chatRepo.GetParticipantIDs(chatID)
	if isSelfChat(ids, userID) {
		return errInvalid("нельзя архивировать чат «Избранное»")
	}
	if err := s.chatRepo.ArchiveChat(chatID, userID); err != nil {
		return errInternal("не удалось архивировать чат")
	}
	return nil
}

func (s *Server) UnarchiveChat(ctx context.Context, chatID, userID uint) error {
	if err := s.requireParticipant(chatID, userID); err != nil {
		return err
	}
	if err := s.chatRepo.UnarchiveChat(chatID, userID); err != nil {
		return errInternal("не удалось разархивировать чат")
	}
	return nil
}

func (s *Server) GetUserArchivedChats(ctx context.Context, userID uint) ([]ChatResponse, error) {
	chats, err := s.chatRepo.FindByUserIDArchived(userID)
	if err != nil {
		return nil, errInternal("не удалось получить архивные чаты")
	}
	var res []ChatResponse
	for _, c := range chats {
		ids, err := s.chatRepo.GetParticipantIDs(c.ID)
		if err != nil {
			continue
		}
		if len(ids) == 2 && contains(ids, s.botID) {
			continue
		}
		if isSelfChat(ids, userID) {
			continue
		}
		pc, err := s.loadChatResponse(ctx, c.ID, userID)
		if err != nil {
			continue
		}
		res = append(res, *pc)
	}
	return res, nil
}

func (s *Server) AddReaction(ctx context.Context, messageID, userID uint, reaction string) (*entity.MessageReaction, error) {
	msg, err := s.messageRepo.FindByID(messageID)
	if err != nil {
		return nil, errNotFound("сообщение не найдено")
	}
	if err := s.requireParticipant(msg.ChatID, userID); err != nil {
		return nil, err
	}
	r := &entity.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Reaction:  reaction,
		CreatedAt: time.Now(),
	}
	if err := s.reactionRepo.Add(r); err != nil {
		return nil, errInternal("не удалось добавить реакцию")
	}
	s.producer.Publish(nats.TopicReactionAdded, itoa(uint64(msg.ChatID)), nats.EventReactionAdded{
		ChatID: int64(msg.ChatID), MessageID: int64(r.MessageID), Reaction: nats.ReactionDTO{
			MessageID: int64(r.MessageID), UserID: int64(r.UserID), Reaction: r.Reaction,
			CreatedAtMs: r.CreatedAt.UnixMilli(),
		},
	})
	return r, nil
}

func (s *Server) RemoveReaction(ctx context.Context, messageID, userID uint, reaction string) error {
	if err := s.reactionRepo.Remove(messageID, userID, reaction); err != nil {
		return errInternal("не удалось удалить реакцию")
	}
	s.producer.Publish(nats.TopicReactionRemoved, itoa(uint64(messageID)), nats.EventReactionRemoved{
		MessageID: int64(messageID), UserID: int64(userID), Reaction: reaction,
	})
	return nil
}

func (s *Server) ForwardMessage(ctx context.Context, chatID, senderID, sourceMessageID uint) (*entity.Message, error) {
	src, err := s.messageRepo.FindByID(sourceMessageID)
	if err != nil {
		return nil, errNotFound("сообщение не найдено")
	}
	if src.SystemType != "" {
		return nil, errInvalid("нельзя переслать системное сообщение")
	}
	if err := s.requireParticipant(chatID, senderID); err != nil {
		return nil, err
	}
	msg := &entity.Message{
		ChatID:                 chatID,
		SenderID:               senderID,
		Text:                   src.Text,
		AttachmentType:         src.AttachmentType,
		AttachmentURL:          src.AttachmentURL,
		AttachmentName:         src.AttachmentName,
		AttachmentSize:         src.AttachmentSize,
		IsForwarded:            true,
		ForwardedFromSenderID:  src.SenderID,
		ForwardedFromChatID:    src.ChatID,
		ForwardedFromMessageID: src.ID,
	}
	if err := s.messageRepo.Create(msg); err != nil {
		return nil, errInternal("не удалось сохранить сообщение")
	}
	s.publishCreated(chatID, msg, nil)
	return msg, nil
}

func (s *Server) GetReactions(ctx context.Context, messageID uint) ([]entity.MessageReaction, error) {
	list, err := s.reactionRepo.FindByMessageID(messageID)
	if err != nil {
		return nil, errInternal("не удалось получить реакции")
	}
	return list, nil
}

// --- helpers ---

func (s *Server) requireParticipant(chatID, userID uint) error {
	ok, err := s.chatRepo.IsParticipant(chatID, userID)
	if err != nil {
		return errInternal("проверка участников не удалась")
	}
	if !ok {
		return errNotFound("чат не найден")
	}
	return nil
}

func (s *Server) loadChatResponse(ctx context.Context, chatID, viewerID uint) (*ChatResponse, error) {
	chat, err := s.chatRepo.FindByID(chatID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, errNotFound("чат не найден")
		}
		return nil, errInternal("не удалось загрузить чат")
	}
	ok, err := s.chatRepo.IsParticipant(chatID, viewerID)
	if err != nil || !ok {
		return nil, errNotFound("чат не найден")
	}

	pids, _ := s.chatRepo.GetParticipantIDs(chatID)
	participants := make([]uint, 0, len(pids))
	for _, p := range pids {
		if p == viewerID {
			continue
		}
		participants = append(participants, p)
	}
	unread, _ := s.chatRepo.GetUnreadCount(chatID, viewerID)
	last, _ := s.messageRepo.GetLastMessage(chatID)

	res := &ChatResponse{
		ID:              chat.ID,
		Name:            chat.Name,
		Type:            chat.Type,
		Avatar:          chat.Avatar,
		PinnedMessageID: chat.PinnedMessageID,
		Participants:    participants,
		UnreadCount:     unread,
		CreatedAt:       chat.CreatedAt,
		UpdatedAt:       chat.UpdatedAt,
		IsFavorites:     isSelfChat(pids, viewerID),
	}
	if last != nil {
		reactions, _ := s.reactionRepo.FindByMessageID(last.ID)
		res.LastMessage = &MessageWithReactions{Msg: last, Reactions: reactions}
	}
	if chat.PinnedMessageID != nil {
		if pm, err := s.messageRepo.FindByID(*chat.PinnedMessageID); err == nil {
			res.PinnedMessage = &MessageWithReactions{Msg: pm}
		}
	}
	return res, nil
}

func (s *Server) publishCreated(chatID uint, m *entity.Message, reactions []entity.MessageReaction) {
	participants, _ := s.chatRepo.GetParticipantIDs(chatID)
	ids := make([]int64, 0, len(participants))
	hasBot := false
	for _, p := range participants {
		ids = append(ids, int64(p))
		if p == s.botID {
			hasBot = true
		}
	}
	dto := toMessageDTO(m, reactions)
	s.producer.Publish(nats.TopicMessageCreated, itoa(uint64(chatID)), nats.EventMessageCreated{
		ChatID:       int64(chatID),
		Message:      dto,
		Participants: ids,
		HasBot:       hasBot,
		BotID:        int64(s.botID),
	})
	if hasBot && uint64(m.SenderID) != uint64(s.botID) {
		s.producer.Publish(nats.TopicAITrigger, itoa(uint64(chatID)), nats.EventMessageCreated{
			ChatID:       int64(chatID),
			Message:      dto,
			Participants: ids,
			HasBot:       true,
			BotID:        int64(s.botID),
		})
	}
}

func (s *Server) HandleCallEnded(value []byte) {
	var e nats.EventCallEnded
	if err := json.Unmarshal(value, &e); err != nil {
		return
	}
	if e.ChatID <= 0 {
		return
	}
	text, senderID := callEndedSystemText(e)
	msg := &entity.Message{
		ChatID:     uint(e.ChatID),
		SenderID:   senderID,
		Text:       text,
		SystemType: "call",
	}
	if err := s.messageRepo.Create(msg); err != nil {
		return
	}
	s.publishCreated(uint(e.ChatID), msg, nil)
}

// callEndedSystemText builds the human-readable system message and picks a
// sender id for a call that reached a terminal state.
func callEndedSystemText(e nats.EventCallEnded) (string, uint) {
	sender := uint(e.CallerID)
	callLabel := "звонок"
	if e.CallType == "video" {
		callLabel = "видеозвонок"
	}
	switch e.Status {
	case "missed":
		return "Пропущенный " + callLabel, sender
	case "rejected":
		return "Звонок отклонён", uint(e.CalleeID)
	case "cancelled":
		return "Звонок отменён", sender
	default:
		if e.DurationMs > 0 {
			return "Звонок завершён · " + formatCallDuration(e.DurationMs), sender
		}
		return "Звонок завершён", sender
	}
}

func formatCallDuration(ms int64) string {
	total := ms / 1000
	m := total / 60
	s := total % 60
	return strconv.FormatInt(m, 10) + ":" + twoDigits(s)
}

func twoDigits(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

func contains(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func isSelfChat(ids []uint, userID uint) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if id != userID {
			return false
		}
	}
	return true
}

func toMessageDTO(m *entity.Message, reactions []entity.MessageReaction) nats.MessageDTO {
	d := nats.MessageDTO{
		ID:                    int64(m.ID),
		ChatID:                int64(m.ChatID),
		SenderID:              int64(m.SenderID),
		Text:                  m.Text,
		IsRead:                m.IsRead,
		Edited:                m.Edited,
		SystemType:            m.SystemType,
		AttachmentType:        m.AttachmentType,
		AttachmentURL:         m.AttachmentURL,
		AttachmentName:        m.AttachmentName,
		AttachmentSize:        attachmentSizeOrZero(m.AttachmentSize),
		CreatedAtMs:           m.CreatedAt.UnixMilli(),
		ReadAtMs:              msOrZero(m.ReadAt),
		EditedAtMs:            msOrZero(m.EditedAt),
		IsForwarded:           m.IsForwarded,
		ForwardedFromSenderID: int64(m.ForwardedFromSenderID),
		ForwardedFromChatID:   int64(m.ForwardedFromChatID),
	}
	for i := range reactions {
		d.Reactions = append(d.Reactions, nats.ReactionDTO{
			MessageID:   int64(reactions[i].MessageID),
			UserID:      int64(reactions[i].UserID),
			Reaction:    reactions[i].Reaction,
			CreatedAtMs: reactions[i].CreatedAt.UnixMilli(),
		})
	}
	return d
}

func attachmentSizeOrZero(p *int) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

func msOrZero(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

func itoa(n uint64) string {
	return strconv.FormatUint(n, 10)
}
