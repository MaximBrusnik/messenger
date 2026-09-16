package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/user"

	"messengermax/userservice/internal/entity"
	"messengermax/userservice/internal/service"
)

type Server struct {
	pb.UnimplementedUserServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.UserProfile, error) {
	v, err := s.svc.GetProfile(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(v), nil
}

func (s *Server) GetProfilesBulk(ctx context.Context, req *pb.GetProfilesBulkRequest) (*pb.GetProfilesBulkResponse, error) {
	if len(req.UserIds) == 0 {
		return &pb.GetProfilesBulkResponse{}, nil
	}
	ids := make([]uint, len(req.UserIds))
	for i, id := range req.UserIds {
		ids[i] = uint(id)
	}
	views, err := s.svc.GetProfilesBulk(ctx, ids)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*pb.UserProfile, 0, len(views))
	for _, v := range views {
		out = append(out, toProto(v))
	}
	return &pb.GetProfilesBulkResponse{Users: out}, nil
}

func (s *Server) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.UsersResponse, error) {
	list, err := s.svc.GetAllUsers(ctx, uint(req.ExcludeId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.UsersResponse{Users: toProtoList(list)}, nil
}

func (s *Server) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.UsersResponse, error) {
	list, err := s.svc.SearchUsers(ctx, req.Query, uint(req.ExcludeId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.UsersResponse{Users: toProtoList(list)}, nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserProfile, error) {
	u := service.ProfileUpdate{
		Username:    req.Username,
		Email:       req.Email,
		Avatar:      req.Avatar,
		Bio:         req.Bio,
		DateOfBirth: req.DateOfBirth,
		ClearAvatar: req.ClearAvatar,
		IsAdmin:     req.IsAdmin,
		IsBot:       req.IsBot,
	}
	v, err := s.svc.UpdateProfile(ctx, uint(req.UserId), u)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(v), nil
}

func (s *Server) ChangePassword(ctx context.Context, _ *pb.ChangePasswordRequest) (*pb.Empty, error) {
	// Password is owned by authservice; userservice is a projection.
	return nil, status.Error(codes.Unimplemented, "password is managed by authservice")
}

func (s *Server) GetContacts(ctx context.Context, req *pb.GetContactsRequest) (*pb.UsersResponse, error) {
	list, err := s.svc.GetContacts(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.UsersResponse{Users: toProtoList(list)}, nil
}

func (s *Server) AddContact(ctx context.Context, req *pb.AddContactRequest) (*pb.UsersResponse, error) {
	if err := s.svc.AddContact(ctx, uint(req.UserId), uint(req.ContactId)); err != nil {
		return nil, toGRPCError(err)
	}
	list, err := s.svc.GetContacts(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.UsersResponse{Users: toProtoList(list)}, nil
}

func (s *Server) GetSettings(ctx context.Context, req *pb.GetSettingsRequest) (*pb.Settings, error) {
	p, err := s.svc.GetSettings(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toSettings(p), nil
}

func (s *Server) UpdateSettings(ctx context.Context, req *pb.UpdateSettingsRequest) (*pb.Settings, error) {
	p, err := s.svc.UpdateSettings(ctx, uint(req.UserId), req.ShowOnlineStatus, req.LastSeenPrivacy, req.AvatarPrivacy)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toSettings(p), nil
}

func (s *Server) ResolveUserByName(ctx context.Context, req *pb.ResolveUserByNameRequest) (*pb.ResolveUserByNameResponse, error) {
	id, found, err := s.svc.ResolveUserByName(ctx, req.Username)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.ResolveUserByNameResponse{UserId: uint64(id), Found: found}, nil
}

func toProto(v service.ProfileView) *pb.UserProfile {
	p := &v.Profile
	statusStr := "offline"
	if v.Online && p.ShowOnlineStatus {
		statusStr = "online"
	}
	dob := ""
	if p.DateOfBirth != nil {
		dob = p.DateOfBirth.Format("2006-01-02")
	}
	return &pb.UserProfile{
		Id:            uint64(p.ID),
		Username:      p.Username,
		Email:         p.Email,
		EmailVerified: p.EmailVerified,
		Avatar:        p.Avatar,
		Status:        statusStr,
		Bio:           p.Bio,
		DateOfBirth:   dob,
		IsBot:         p.IsBot,
		IsAdmin:       p.IsAdmin,
		CreatedAt:     timestamppb.New(p.CreatedAt),
	}
}

func toProtoList(profiles []entity.Profile) []*pb.UserProfile {
	out := make([]*pb.UserProfile, 0, len(profiles))
	for i := range profiles {
		out = append(out, toProto(service.ProfileView{Profile: profiles[i]}))
	}
	return out
}

func toSettings(p *entity.Profile) *pb.Settings {
	return &pb.Settings{
		ShowOnlineStatus: p.ShowOnlineStatus,
		LastSeenPrivacy:  p.LastSeenPrivacy,
		AvatarPrivacy:    p.AvatarPrivacy,
	}
}

func toGRPCError(err error) error {
	return status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
}
