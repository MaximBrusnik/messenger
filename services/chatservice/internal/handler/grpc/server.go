package grpc

import (
	"context"
	"errors"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/chat"

	"messengermax/chatservice/internal/entity"
	"messengermax/chatservice/internal/service"
)

type Server struct {
	pb.UnimplementedChatServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.Chat, error) {
	participantIDs := make([]uint, 0, len(req.ParticipantIds))
	for _, p := range req.ParticipantIds {
		participantIDs = append(participantIDs, uint(p))
	}
	resp, err := s.svc.CreateChat(ctx, uint(req.UserId), req.Type, req.Name, participantIDs)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toPbChat(resp), nil
}

func (s *Server) GetUserChats(ctx context.Context, req *pb.GetUserChatsRequest) (*pb.ChatsResponse, error) {
	chats, err := s.svc.GetUserChats(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	res := &pb.ChatsResponse{Chats: make([]*pb.Chat, 0, len(chats))}
	for i := range chats {
		res.Chats = append(res.Chats, toPbChat(&chats[i]))
	}
	return res, nil
}

func (s *Server) GetChatByID(ctx context.Context, req *pb.GetChatRequest) (*pb.Chat, error) {
	resp, err := s.svc.GetChat(ctx, uint(req.ChatId), uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toPbChat(resp), nil
}

func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.Message, error) {
	msg, err := s.svc.SendMessage(ctx, uint(req.ChatId), uint(req.UserId), req.Content, req.AttachmentUrl, req.AttachmentType, req.AttachmentName, req.AttachmentSize)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toMessageProto(msg, nil), nil
}

func (s *Server) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.MessagesResponse, error) {
	messages, err := s.svc.GetMessages(ctx, uint(req.ChatId), uint(req.UserId), int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, toGRPCError(err)
	}
	res := &pb.MessagesResponse{Messages: make([]*pb.Message, 0, len(messages))}
	for _, m := range messages {
		res.Messages = append(res.Messages, toMessageProto(m.Msg, m.Reactions))
	}
	return res, nil
}

func (s *Server) EditMessage(ctx context.Context, req *pb.EditMessageRequest) (*pb.Message, error) {
	msg, err := s.svc.EditMessage(ctx, uint(req.UserId), uint(req.MessageId), req.Content)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toMessageProto(msg, nil), nil
}

func (s *Server) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.Empty, error) {
	if err := s.svc.DeleteMessage(ctx, uint(req.ChatId), uint(req.UserId), uint(req.MessageId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.Empty, error) {
	if err := s.svc.MarkAsRead(ctx, uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) PinMessage(ctx context.Context, req *pb.PinMessageRequest) (*pb.Message, error) {
	msg, reactions, err := s.svc.PinMessage(ctx, uint(req.ChatId), uint(req.UserId), uint(req.MessageId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toMessageProto(msg, reactions), nil
}

func (s *Server) UnpinMessage(ctx context.Context, req *pb.UnpinMessageRequest) (*pb.Empty, error) {
	if err := s.svc.UnpinMessage(ctx, uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*pb.Empty, error) {
	if err := s.svc.DeleteChat(ctx, uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) GetOrCreateAIChat(ctx context.Context, req *pb.GetOrCreateAIChatRequest) (*pb.Chat, error) {
	resp, err := s.svc.GetOrCreateAIChat(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toPbChat(resp), nil
}

func (s *Server) GetOrCreateFavoritesChat(ctx context.Context, req *pb.GetOrCreateFavoritesChatRequest) (*pb.Chat, error) {
	resp, err := s.svc.GetOrCreateFavoritesChat(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toPbChat(resp), nil
}

func (s *Server) ArchiveChat(ctx context.Context, req *pb.ArchiveChatRequest) (*pb.Empty, error) {
	if err := s.svc.ArchiveChat(ctx, uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) UnarchiveChat(ctx context.Context, req *pb.UnarchiveChatRequest) (*pb.Empty, error) {
	if err := s.svc.UnarchiveChat(ctx, uint(req.ChatId), uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) GetUserArchivedChats(ctx context.Context, req *pb.GetUserChatsRequest) (*pb.ChatsResponse, error) {
	chats, err := s.svc.GetUserArchivedChats(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	res := &pb.ChatsResponse{Chats: make([]*pb.Chat, 0, len(chats))}
	for i := range chats {
		res.Chats = append(res.Chats, toPbChat(&chats[i]))
	}
	return res, nil
}

func (s *Server) AddReaction(ctx context.Context, req *pb.AddReactionRequest) (*pb.Reaction, error) {
	r, err := s.svc.AddReaction(ctx, uint(req.MessageId), uint(req.UserId), req.Reaction)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toReactionProto(r, ""), nil
}

func (s *Server) RemoveReaction(ctx context.Context, req *pb.RemoveReactionRequest) (*pb.Empty, error) {
	if err := s.svc.RemoveReaction(ctx, uint(req.MessageId), uint(req.UserId), req.Reaction); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) ForwardMessage(ctx context.Context, req *pb.ForwardMessageRequest) (*pb.Message, error) {
	msg, err := s.svc.ForwardMessage(ctx, uint(req.ChatId), uint(req.UserId), uint(req.MessageId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toMessageProto(msg, nil), nil
}

func (s *Server) GetReactions(ctx context.Context, req *pb.GetReactionsRequest) (*pb.ReactionsResponse, error) {
	list, err := s.svc.GetReactions(ctx, uint(req.MessageId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	res := &pb.ReactionsResponse{Reactions: make([]*pb.Reaction, 0, len(list))}
	for i := range list {
		res.Reactions = append(res.Reactions, toReactionProto(&list[i], ""))
	}
	return res, nil
}

func toPbChat(resp *service.ChatResponse) *pb.Chat {
	ch := &pb.Chat{
		Id:              uint64(resp.ID),
		Name:            resp.Name,
		Type:            resp.Type,
		Avatar:          resp.Avatar,
		PinnedMessageId: uint64val(resp.PinnedMessageID),
		Participants:    make([]*pb.ChatUser, 0, len(resp.Participants)),
		Unread:          int32(resp.UnreadCount),
		CreatedAt:       timestamppb.New(resp.CreatedAt),
		UpdatedAt:       timestamppb.New(resp.UpdatedAt),
		IsFavorites:     resp.IsFavorites,
	}
	for _, p := range resp.Participants {
		ch.Participants = append(ch.Participants, &pb.ChatUser{UserId: uint64(p)})
	}
	if resp.LastMessage != nil {
		ch.LastMessage = toMessageProto(resp.LastMessage.Msg, resp.LastMessage.Reactions)
	}
	if resp.PinnedMessage != nil {
		ch.PinnedMessage = toMessageProto(resp.PinnedMessage.Msg, resp.PinnedMessage.Reactions)
	}
	return ch
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

func uint64val(p *uint) uint64 {
	if p == nil {
		return 0
	}
	return uint64(*p)
}

func toGRPCError(err error) error {
	code := codes.Internal
	msg := err.Error()
	var e *apperr.Error
	if errors.As(err, &e) {
		code = e.Code
		msg = e.Message
	}
	return status.Error(code, msg)
}
