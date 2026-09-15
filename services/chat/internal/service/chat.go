package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/chat/internal/entity"
	"messengermax/chat/internal/repo"
	"messengermax/pkg/nats"
	pb "messengermax/proto/gen/chat"
)

// BotUsername is the well-known AI assistant username seeded in auth-service.
const BotUsername = "Ассистент"

type Server struct {
	pb.UnimplementedChatServiceServer
	chatRepo     repo.ChatRepository
	messageRepo  repo.MessageRepository
	reactionRepo repo.ReactionRepository
	producer     *nats.Producer
	botID        uint
}

func NewServer(
	chatRepo repo.ChatRepository,
	messageRepo repo.MessageRepository,
	reactionRepo repo.ReactionRepository,
	producer *nats.Producer,
	botID uint,
) *Server {
	return &Server{
		chatRepo:     chatRepo,
		messageRepo:  messageRepo,
		reactionRepo: reactionRepo,
		producer:     producer,
		botID:        botID,
	}
}

func (s *Server) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.Chat, error) {
	creator := uint(req.UserId)
	if req.Type == "" {
		req.Type = "private"
	}
	if req.Type == "private" {
		if len(req.ParticipantIds) != 1 {
			return nil, status.Error(codes.InvalidArgument, "личный чат требует одного участника")
		}
		if existing, err := s.chatRepo.FindPrivateChat(creator, uint(req.ParticipantIds[0])); err == nil {
			return s.loadChatResponse(ctx, existing.ID, creator)
		}
	}

	chat := &entity.Chat{Name: req.Name, Type: req.Type}
	ids := make([]uint, 0, len(req.ParticipantIds)+1)
	ids = append(ids, creator)
	for _, p := range req.ParticipantIds {
		ids = append(ids, uint(p))
	}
	if err := s.chatRepo.Create(chat, ids); err != nil {
		return nil, status.Error(codes.Internal, "не удалось создать чат")
	}
	return s.loadChatResponse(ctx, chat.ID, creator)
}

func (s *Server) GetUserChats(ctx context.Context, req *pb.GetUserChatsRequest) (*pb.ChatsResponse, error) {
	chats, err := s.chatRepo.FindByUserID(uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить чаты")
	}
	var res []*pb.Chat
	for _, c := range chats {
		// skip AI bot private chats in the main list
		ids, err := s.chatRepo.GetParticipantIDs(c.ID)
		if err != nil {
			continue
		}
		if len(ids) == 2 && contains(ids, s.botID) {
			continue
		}
		pc, err := s.loadChatResponse(ctx, c.ID, uint(req.UserId))
		if err != nil {
			continue
		}
		res = append(res, pc)
	}
	return &pb.ChatsResponse{Chats: res}, nil
}

func (s *Server) GetChatByID(ctx context.Context, req *pb.GetChatRequest) (*pb.Chat, error) {
	return s.loadChatResponse(ctx, uint(req.ChatId), uint(req.UserId))
}

func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.Message, error) {
	if err := s.requireParticipant(uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, err
	}
	if req.Content == "" && req.AttachmentUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "сообщение пустое")
	}
	var size *int
	if req.AttachmentSize > 0 {
		n := int(req.AttachmentSize)
		size = &n
	}
	msg := &entity.Message{
		ChatID:         uint(req.ChatId),
		SenderID:       uint(req.UserId),
		Text:           req.Content,
		AttachmentType: req.AttachmentType,
		AttachmentURL:  req.AttachmentUrl,
		AttachmentName: req.AttachmentName,
		AttachmentSize: size,
	}
	if err := s.messageRepo.Create(msg); err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить сообщение")
	}

	m := toMessageProto(msg, nil)
	s.publishCreated(uint(req.ChatId), m)
	return m, nil
}

func (s *Server) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.MessagesResponse, error) {
	if err := s.requireParticipant(uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, err
	}
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	messages, err := s.messageRepo.FindByChatID(uint(req.ChatId), limit, int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить сообщения")
	}
	res := make([]*pb.Message, 0, len(messages))
	for i := range messages {
		reactions, _ := s.reactionRepo.FindByMessageID(messages[i].ID)
		res = append(res, toMessageProto(&messages[i], reactions))
	}
	return &pb.MessagesResponse{Messages: res}, nil
}

func (s *Server) EditMessage(ctx context.Context, req *pb.EditMessageRequest) (*pb.Message, error) {
	msg, err := s.messageRepo.FindByID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "сообщение не найдено")
	}
	if msg.SenderID != uint(req.UserId) {
		return nil, status.Error(codes.PermissionDenied, "нельзя редактировать чужое сообщение")
	}
	if err := s.messageRepo.UpdateText(msg.ID, req.Content); err != nil {
		return nil, status.Error(codes.Internal, "не удалось изменить сообщение")
	}
	fresh, err := s.messageRepo.FindByID(msg.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось загрузить сообщение")
	}
	m := toMessageProto(fresh, nil)
	s.producer.Publish(nats.TopicMessageEdited, itoa(uint64(msg.ChatID)), nats.EventMessageEdited{
		ChatID:  int64(fresh.ChatID),
		Message: toMessageDTO(m),
	})
	return m, nil
}

func (s *Server) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.Empty, error) {
	msg, err := s.messageRepo.FindByID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "сообщение не найдено")
	}
	if msg.ChatID != uint(req.ChatId) {
		return nil, status.Error(codes.InvalidArgument, "неверный чат")
	}
	if msg.SenderID != uint(req.UserId) {
		return nil, status.Error(codes.PermissionDenied, "нельзя удалить чужое сообщение")
	}
	if err := s.messageRepo.Delete(msg.ID); err != nil {
		return nil, status.Error(codes.Internal, "не удалось удалить сообщение")
	}
	s.producer.Publish(nats.TopicMessageDeleted, itoa(uint64(msg.ChatID)), nats.EventMessageDeleted{
		ChatID: int64(msg.ChatID), MessageID: int64(msg.ID),
	})
	return &pb.Empty{}, nil
}

func (s *Server) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.Empty, error) {
	if err := s.requireParticipant(uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, err
	}
	ids, err := s.messageRepo.MarkChatAsRead(uint(req.ChatId), uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось отметить прочитанным")
	}
	if len(ids) > 0 {
		_ = s.chatRepo.UpdateLastRead(uint(req.ChatId), uint(req.UserId))
		idss := make([]int64, 0, len(ids))
		for _, id := range ids {
			idss = append(idss, int64(id))
		}
		s.producer.Publish(nats.TopicReadReceived, itoa(req.ChatId), nats.EventMessagesRead{
			ChatID: int64(req.ChatId), UserID: int64(req.UserId), MessageIDs: idss,
		})
	}
	return &pb.Empty{}, nil
}

func (s *Server) PinMessage(ctx context.Context, req *pb.PinMessageRequest) (*pb.Message, error) {
	msg, err := s.messageRepo.FindByID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "сообщение не найдено")
	}
	if msg.ChatID != uint(req.ChatId) {
		return nil, status.Error(codes.InvalidArgument, "неверный чат")
	}
	if err := s.chatRepo.PinMessage(uint(req.ChatId), msg.ID); err != nil {
		return nil, status.Error(codes.Internal, "не удалось закрепить")
	}
	system := &entity.Message{ChatID: msg.ChatID, SenderID: uint(req.UserId), Text: "", SystemType: "pin"}
	_ = s.messageRepo.Create(system)
	reactions, _ := s.reactionRepo.FindByMessageID(msg.ID)
	s.producer.Publish(nats.TopicMessagePinned, itoa(req.ChatId), nats.EventMessagePinned{
		ChatID:  int64(req.ChatId),
		Message: toMessageDTO(toMessageProto(msg, reactions)),
	})
	return toMessageProto(msg, reactions), nil
}

func (s *Server) UnpinMessage(ctx context.Context, req *pb.UnpinMessageRequest) (*pb.Empty, error) {
	if err := s.chatRepo.UnpinMessage(uint(req.ChatId)); err != nil {
		return nil, status.Error(codes.Internal, "не удалось открепить")
	}
	system := &entity.Message{ChatID: uint(req.ChatId), SenderID: uint(req.UserId), Text: "", SystemType: "unpin"}
	_ = s.messageRepo.Create(system)
	s.producer.Publish(nats.TopicMessageUnpinned, itoa(req.ChatId), nats.EventMessagePinned{
		ChatID: int64(req.ChatId),
	})
	return &pb.Empty{}, nil
}

// HandleCallEnded consumes calls.call.ended events and writes a system
// message ("call") into the chat the call originated from.
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
	m := toMessageProto(msg, nil)
	s.publishCreated(uint(e.ChatID), m)
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

func (s *Server) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*pb.Empty, error) {
	if err := s.requireParticipant(uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, err
	}
	ids, _ := s.chatRepo.GetParticipantIDs(uint(req.ChatId))
	if err := s.chatRepo.Delete(uint(req.ChatId)); err != nil {
		return nil, status.Error(codes.Internal, "не удалось удалить чат")
	}
	users := make([]int64, 0, len(ids))
	for _, id := range ids {
		users = append(users, int64(id))
	}
	s.producer.Publish(nats.TopicChatDeleted, itoa(req.ChatId), nats.EventChatDeleted{
		ChatID: int64(req.ChatId), Users: users,
	})
	return &pb.Empty{}, nil
}

func (s *Server) GetOrCreateAIChat(ctx context.Context, req *pb.GetOrCreateAIChatRequest) (*pb.Chat, error) {
	if s.botID == 0 {
		return nil, status.Error(codes.Unavailable, "AI не настроен")
	}
	if existing, err := s.chatRepo.FindPrivateChat(uint(req.UserId), s.botID); err == nil {
		return s.loadChatResponse(ctx, existing.ID, uint(req.UserId))
	}
	chat := &entity.Chat{Type: "private"}
	if err := s.chatRepo.Create(chat, []uint{uint(req.UserId), s.botID}); err != nil {
		return nil, status.Error(codes.Internal, "не удалось создать AI-чат")
	}
	return s.loadChatResponse(ctx, chat.ID, uint(req.UserId))
}

func (s *Server) AddReaction(ctx context.Context, req *pb.AddReactionRequest) (*pb.Reaction, error) {
	msg, err := s.messageRepo.FindByID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "сообщение не найдено")
	}
	if err := s.requireParticipant(msg.ChatID, uint(req.UserId)); err != nil {
		return nil, err
	}
	r := &entity.MessageReaction{
		MessageID: uint(req.MessageId),
		UserID:    uint(req.UserId),
		Reaction:  req.Reaction,
		CreatedAt: time.Now(),
	}
	if err := s.reactionRepo.Add(r); err != nil {
		return nil, status.Error(codes.Internal, "не удалось добавить реакцию")
	}
	protoReaction := toReactionProto(r, "")
	s.producer.Publish(nats.TopicReactionAdded, itoa(uint64(msg.ChatID)), nats.EventReactionAdded{
		ChatID: int64(msg.ChatID), MessageID: int64(r.MessageID), Reaction: nats.ReactionDTO{
			MessageID: int64(r.MessageID), UserID: int64(r.UserID), Reaction: r.Reaction,
			CreatedAtMs: r.CreatedAt.UnixMilli(),
		},
	})
	return protoReaction, nil
}

func (s *Server) RemoveReaction(ctx context.Context, req *pb.RemoveReactionRequest) (*pb.Empty, error) {
	if err := s.reactionRepo.Remove(uint(req.MessageId), uint(req.UserId), req.Reaction); err != nil {
		return nil, status.Error(codes.Internal, "не удалось удалить реакцию")
	}
	s.producer.Publish(nats.TopicReactionRemoved, itoa(req.MessageId), nats.EventReactionRemoved{
		MessageID: int64(req.MessageId), UserID: int64(req.UserId), Reaction: req.Reaction,
	})
	return &pb.Empty{}, nil
}

func (s *Server) ForwardMessage(ctx context.Context, req *pb.ForwardMessageRequest) (*pb.Message, error) {
	src, err := s.messageRepo.FindByID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "сообщение не найдено")
	}
	if src.SystemType != "" {
		return nil, status.Error(codes.InvalidArgument, "нельзя переслать системное сообщение")
	}
	if err := s.requireParticipant(uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, err
	}
	msg := &entity.Message{
		ChatID:                 uint(req.ChatId),
		SenderID:               uint(req.UserId),
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
		return nil, status.Error(codes.Internal, "не удалось сохранить сообщение")
	}
	m := toMessageProto(msg, nil)
	s.publishCreated(uint(req.ChatId), m)
	return m, nil
}

func (s *Server) GetReactions(ctx context.Context, req *pb.GetReactionsRequest) (*pb.ReactionsResponse, error) {
	list, err := s.reactionRepo.FindByMessageID(uint(req.MessageId))
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить реакции")
	}
	res := make([]*pb.Reaction, 0, len(list))
	for i := range list {
		res = append(res, toReactionProto(&list[i], ""))
	}
	return &pb.ReactionsResponse{Reactions: res}, nil
}

// --- helpers ---

func (s *Server) requireParticipant(chatID, userID uint) error {
	ok, err := s.chatRepo.IsParticipant(chatID, userID)
	if err != nil {
		return status.Error(codes.Internal, "проверка участников не удалась")
	}
	if !ok {
		return status.Error(codes.NotFound, "чат не найден")
	}
	return nil
}

func (s *Server) loadChatResponse(ctx context.Context, chatID, viewerID uint) (*pb.Chat, error) {
	chat, err := s.chatRepo.FindByID(chatID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "чат не найден")
		}
		return nil, status.Error(codes.Internal, "не удалось загрузить чат")
	}
	ok, err := s.chatRepo.IsParticipant(chatID, viewerID)
	if err != nil || !ok {
		return nil, status.Error(codes.NotFound, "чат не найден")
	}

	pids, _ := s.chatRepo.GetParticipantIDs(chatID)
	participants := make([]*pb.ChatUser, 0, len(pids))
	for _, p := range pids {
		if p == viewerID {
			continue
		}
		participants = append(participants, &pb.ChatUser{UserId: uint64(p)})
	}
	unread, _ := s.chatRepo.GetUnreadCount(chatID, viewerID)
	last, _ := s.messageRepo.GetLastMessage(chatID)

	res := &pb.Chat{
		Id:              uint64(chat.ID),
		Name:            chat.Name,
		Type:            chat.Type,
		Avatar:          chat.Avatar,
		PinnedMessageId: uint64val(chat.PinnedMessageID),
		Participants:    participants,
		Unread:          int32(unread),
		CreatedAt:       timestamppb.New(chat.CreatedAt),
		UpdatedAt:       timestamppb.New(chat.UpdatedAt),
	}
	if last != nil {
		reactions, _ := s.reactionRepo.FindByMessageID(last.ID)
		res.LastMessage = toMessageProto(last, reactions)
	}
	if chat.PinnedMessageID != nil {
		if pm, err := s.messageRepo.FindByID(*chat.PinnedMessageID); err == nil {
			res.PinnedMessage = toMessageProto(pm, nil)
		}
	}
	return res, nil
}

func (s *Server) publishCreated(chatID uint, m *pb.Message) {
	participants, _ := s.chatRepo.GetParticipantIDs(chatID)
	ids := make([]int64, 0, len(participants))
	hasBot := false
	for _, p := range participants {
		ids = append(ids, int64(p))
		if p == s.botID {
			hasBot = true
		}
	}
	s.producer.Publish(nats.TopicMessageCreated, itoa(uint64(chatID)), nats.EventMessageCreated{
		ChatID:       int64(chatID),
		Message:      toMessageDTO(m),
		Participants: ids,
		HasBot:       hasBot,
		BotID:        int64(s.botID),
	})
	if hasBot && int64(m.SenderId) != int64(s.botID) {
		s.producer.Publish(nats.TopicAITrigger, itoa(uint64(chatID)), nats.EventMessageCreated{
			ChatID:       int64(chatID),
			Message:      toMessageDTO(m),
			Participants: ids,
			HasBot:       true,
			BotID:        int64(s.botID),
		})
	}
}

func contains(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func toMessageDTO(m *pb.Message) nats.MessageDTO {
	d := nats.MessageDTO{
		ID:                    int64(m.Id),
		ChatID:                int64(m.ChatId),
		SenderID:              int64(m.SenderId),
		Text:                  m.Text,
		IsRead:                m.IsRead,
		Edited:                m.Edited,
		SystemType:            m.SystemType,
		AttachmentType:        m.AttachmentType,
		AttachmentURL:         m.AttachmentUrl,
		AttachmentName:        m.AttachmentName,
		AttachmentSize:        m.AttachmentSize,
		CreatedAtMs:           m.GetCreatedAt().GetSeconds() * 1000,
		ReadAtMs:              m.GetReadAt().GetSeconds() * 1000,
		EditedAtMs:            m.GetEditedAt().GetSeconds() * 1000,
		IsForwarded:           m.IsForwarded,
		ForwardedFromSenderID: int64(m.ForwardedFromSenderId),
		ForwardedFromChatID:   int64(m.ForwardedFromChatId),
	}
	for _, r := range m.Reactions {
		d.Reactions = append(d.Reactions, nats.ReactionDTO{
			MessageID:   int64(r.MessageId),
			UserID:      int64(r.UserId),
			Reaction:    r.Reaction,
			Username:    r.Username,
			CreatedAtMs: r.GetCreatedAt().GetSeconds() * 1000,
		})
	}
	return d
}

func toMessageProto(m *entity.Message, reactions []entity.MessageReaction) *pb.Message {
	res := &pb.Message{
		Id:             uint64(m.ID),
		ChatId:         uint64(m.ChatID),
		SenderId:       uint64(m.SenderID),
		Text:           m.Text,
		IsRead:         m.IsRead,
		Edited:         m.Edited,
		SystemType:     m.SystemType,
		AttachmentType: m.AttachmentType,
		AttachmentUrl:  m.AttachmentURL,
		AttachmentName: m.AttachmentName,
		CreatedAt:      timestamppb.New(m.CreatedAt),
		IsForwarded:    m.IsForwarded,
	}
	if m.ForwardedFromSenderID > 0 {
		res.ForwardedFromSenderId = uint64(m.ForwardedFromSenderID)
	}
	if m.ForwardedFromChatID > 0 {
		res.ForwardedFromChatId = uint64(m.ForwardedFromChatID)
	}
	if m.ForwardedFromMessageID > 0 {
		res.ForwardedFromMessageId = uint64(m.ForwardedFromMessageID)
	}
	if m.ReadAt != nil {
		res.ReadAt = timestamppb.New(*m.ReadAt)
	}
	if m.EditedAt != nil {
		res.EditedAt = timestamppb.New(*m.EditedAt)
	}
	if m.AttachmentSize != nil {
		res.AttachmentSize = int64(*m.AttachmentSize)
	}
	if reactions != nil {
		for i := range reactions {
			res.Reactions = append(res.Reactions, toReactionProto(&reactions[i], ""))
		}
		sort.Slice(res.Reactions, func(x, y int) bool {
			return res.Reactions[x].UserId < res.Reactions[y].UserId
		})
	}
	return res
}

func toReactionProto(r *entity.MessageReaction, username string) *pb.Reaction {
	return &pb.Reaction{
		MessageId: uint64(r.MessageID),
		UserId:    uint64(r.UserID),
		Reaction:  r.Reaction,
		Username:  username,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
}

func itoa(n uint64) string {
	return strconv.FormatUint(n, 10)
}

func uint64val(p *uint) uint64 {
	if p == nil {
		return 0
	}
	return uint64(*p)
}
