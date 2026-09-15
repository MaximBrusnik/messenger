package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedredis "messengermax/pkg/redis"
	pb "messengermax/proto/gen/user"

	"messengermax/user/internal/entity"
	"messengermax/user/internal/repo"
)

type Server struct {
	pb.UnimplementedUserServiceServer
	profileRepo repo.ProfileRepository
	redis       *sharedredis.Client
}

func NewServer(profileRepo repo.ProfileRepository, redis *sharedredis.Client) *Server {
	return &Server{profileRepo: profileRepo, redis: redis}
}

func (s *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.UserProfile, error) {
	p, err := s.profileRepo.FindByID(uint(req.UserId))
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return nil, notFound(err)
		}
		// Fallback: profile was not mirrored during registration (e.g. user-service
		// was briefly unavailable). Create an empty profile on first read so it
		// is always present. Non-fatal — enrich at next update.
		p = &entity.Profile{
			ID:       uint(req.UserId),
			Username: fmt.Sprintf("user_%d", req.UserId),
			Email:    fmt.Sprintf("user_%d@placeholder.local", req.UserId),
		}
		if err := s.profileRepo.Create(p); err != nil {
			return nil, status.Error(codes.Internal, "не удалось создать профиль")
		}
		return toProto(p, s.isOnline(ctx, p.ID)), nil
	}
	return toProto(p, s.isOnline(ctx, p.ID)), nil
}

func (s *Server) GetProfilesBulk(ctx context.Context, req *pb.GetProfilesBulkRequest) (*pb.GetProfilesBulkResponse, error) {
	if len(req.UserIds) == 0 {
		return &pb.GetProfilesBulkResponse{}, nil
	}
	var out []*pb.UserProfile
	for _, id := range req.UserIds {
		p, err := s.profileRepo.FindByID(uint(id))
		if err != nil {
			continue
		}
		out = append(out, toProto(p, s.isOnline(ctx, p.ID)))
	}
	return &pb.GetProfilesBulkResponse{Users: out}, nil
}

func (s *Server) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.UsersResponse, error) {
	list, err := s.profileRepo.FindAll(uint(req.ExcludeId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list users")
	}
	return &pb.UsersResponse{Users: s.enrich(list)}, nil
}

func (s *Server) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.UsersResponse, error) {
	list, err := s.profileRepo.Search(req.Query, uint(req.ExcludeId))
	if err != nil {
		return nil, status.Error(codes.Internal, "search failed")
	}
	return &pb.UsersResponse{Users: s.enrich(list)}, nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserProfile, error) {
	p, err := s.profileRepo.FindByID(uint(req.UserId))
	if err != nil {
		p = &entity.Profile{
			ID:       uint(req.UserId),
			Username: req.Username,
			Email:    req.Email,
		}
		if p.Username == "" {
			p.Username = fmt.Sprintf("user_%d", req.UserId)
		}
		if p.Email == "" {
			p.Email = fmt.Sprintf("user_%d@placeholder.local", req.UserId)
		}
		if err := s.profileRepo.Create(p); err != nil {
			return nil, status.Error(codes.Internal, "не удалось создать профиль")
		}
		return toProto(p, false), nil
	}
	if req.Username != "" {
		other, err := s.profileRepo.FindByUsername(req.Username)
		if err == nil && other.ID != p.ID {
			return nil, status.Error(codes.AlreadyExists, "username taken")
		}
		p.Username = req.Username
	}
	if req.Email != "" && req.Email != p.Email {
		p.Email = req.Email
		p.EmailVerified = false
	}
	if req.Avatar != "" {
		p.Avatar = req.Avatar
	} else if req.ClearAvatar {
		p.Avatar = ""
	}
	if req.Bio != "" {
		p.Bio = req.Bio
	}
	if req.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			p.DateOfBirth = &t
		}
	}
	if err := s.profileRepo.Update(p); err != nil {
		return nil, status.Error(codes.Internal, "update failed")
	}
	return toProto(p, s.isOnline(ctx, p.ID)), nil
}

func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.Empty, error) {
	// Password is owned by auth-service; user-service is a projection.
	// Returning InvalidArgument signals the gateway to route elsewhere.
	return nil, status.Error(codes.Unimplemented, "password is managed by auth-service")
}

func (s *Server) GetContacts(ctx context.Context, req *pb.GetContactsRequest) (*pb.UsersResponse, error) {
	list, err := s.profileRepo.GetContacts(uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, "contacts failed")
	}
	return &pb.UsersResponse{Users: s.enrich(list)}, nil
}

func (s *Server) AddContact(ctx context.Context, req *pb.AddContactRequest) (*pb.UsersResponse, error) {
	if err := s.profileRepo.AddContact(uint(req.UserId), uint(req.ContactId)); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return s.GetContacts(ctx, &pb.GetContactsRequest{UserId: req.UserId})
}

func (s *Server) GetSettings(ctx context.Context, req *pb.GetSettingsRequest) (*pb.Settings, error) {
	p, err := s.profileRepo.FindByID(uint(req.UserId))
	if err != nil {
		return nil, notFound(err)
	}
	return &pb.Settings{
		ShowOnlineStatus: p.ShowOnlineStatus,
		LastSeenPrivacy:  p.LastSeenPrivacy,
		AvatarPrivacy:    p.AvatarPrivacy,
	}, nil
}

func (s *Server) UpdateSettings(ctx context.Context, req *pb.UpdateSettingsRequest) (*pb.Settings, error) {
	p, err := s.profileRepo.FindByID(uint(req.UserId))
	if err != nil {
		return nil, notFound(err)
	}
	p.ShowOnlineStatus = req.ShowOnlineStatus
	if req.LastSeenPrivacy != "" {
		p.LastSeenPrivacy = req.LastSeenPrivacy
	}
	if req.AvatarPrivacy != "" {
		p.AvatarPrivacy = req.AvatarPrivacy
	}
	if err := s.profileRepo.Update(p); err != nil {
		return nil, status.Error(codes.Internal, "update failed")
	}
	return &pb.Settings{
		ShowOnlineStatus: p.ShowOnlineStatus,
		LastSeenPrivacy:  p.LastSeenPrivacy,
		AvatarPrivacy:    p.AvatarPrivacy,
	}, nil
}

func (s *Server) ResolveUserByName(ctx context.Context, req *pb.ResolveUserByNameRequest) (*pb.ResolveUserByNameResponse, error) {
	p, err := s.profileRepo.FindByUsername(req.Username)
	if err != nil {
		return &pb.ResolveUserByNameResponse{Found: false}, nil
	}
	return &pb.ResolveUserByNameResponse{UserId: uint64(p.ID), Found: true}, nil
}

func (s *Server) enrich(profiles []entity.Profile) []*pb.UserProfile {
	out := make([]*pb.UserProfile, 0, len(profiles))
	for i := range profiles {
		out = append(out, toProto(&profiles[i], false))
	}
	return out
}

func (s *Server) isOnline(ctx context.Context, userID uint) bool {
	return s.redis.IsOnline(ctx, userID)
}

func toProto(p *entity.Profile, online bool) *pb.UserProfile {
	statusStr := "offline"
	if online && p.ShowOnlineStatus {
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

func notFound(err error) error {
	if errors.Is(err, repo.ErrNotFound) {
		return status.Error(codes.NotFound, "profile not found")
	}
	return status.Error(codes.Internal, err.Error())
}
