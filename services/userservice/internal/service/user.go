package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	sharedredis "messengermax/pkg/redis"

	"messengermax/userservice/internal/entity"
	"messengermax/userservice/internal/repo"
)

type Server struct {
	profileRepo repo.ProfileRepository
	redis       *sharedredis.Client
}

func NewServer(profileRepo repo.ProfileRepository, redis *sharedredis.Client) *Server {
	return &Server{profileRepo: profileRepo, redis: redis}
}

func (s *Server) GetProfile(ctx context.Context, userID, viewerID uint) (ProfileView, error) {
	p, err := s.profileRepo.FindByID(userID)
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return ProfileView{}, errInternal("profile not found")
		}
		p = &entity.Profile{
			ID:       userID,
			Username: fmt.Sprintf("user_%d", userID),
			Email:    fmt.Sprintf("user_%d@placeholder.local", userID),
		}
		if err := s.profileRepo.Create(p); err != nil {
			return ProfileView{}, errInternal("не удалось создать профиль")
		}
		s.hideAvatar(p, viewerID)
		return ProfileView{Profile: *p, Online: false}, nil
	}
	s.hideAvatar(p, viewerID)
	return ProfileView{Profile: *p, Online: s.isOnline(ctx, p.ID)}, nil
}

func (s *Server) GetProfilesBulk(ctx context.Context, ids []uint, viewerID uint) ([]ProfileView, error) {
	profiles, err := s.profileRepo.FindByIDs(ids)
	if err != nil {
		return nil, errInternal("profiles failed")
	}
	out := make([]ProfileView, 0, len(profiles))
	for i := range profiles {
		s.hideAvatar(&profiles[i], viewerID)
		out = append(out, ProfileView{Profile: profiles[i], Online: s.isOnline(ctx, profiles[i].ID)})
	}
	return out, nil
}

func (s *Server) GetAllUsers(ctx context.Context, excludeID uint) ([]entity.Profile, error) {
	list, err := s.profileRepo.FindAll(excludeID)
	if err != nil {
		return nil, errInternal("failed to list users")
	}
	for i := range list {
		s.hideAvatar(&list[i], excludeID)
	}
	return list, nil
}

func (s *Server) SearchUsers(ctx context.Context, query string, excludeID uint) ([]entity.Profile, error) {
	list, err := s.profileRepo.Search(query, excludeID)
	if err != nil {
		return nil, errInternal("search failed")
	}
	for i := range list {
		s.hideAvatar(&list[i], excludeID)
	}
	return list, nil
}

func (s *Server) UpdateProfile(ctx context.Context, userID uint, u ProfileUpdate) (ProfileView, error) {
	p, err := s.profileRepo.FindByID(userID)
	if err != nil {
		p = &entity.Profile{
			ID:       userID,
			Username: u.Username,
			Email:    u.Email,
		}
		if u.IsAdmin != nil {
			p.IsAdmin = *u.IsAdmin
		}
		if u.IsBot != nil {
			p.IsBot = *u.IsBot
		}
		if p.Username == "" {
			p.Username = fmt.Sprintf("user_%d", userID)
		}
		if p.Email == "" {
			p.Email = fmt.Sprintf("user_%d@placeholder.local", userID)
		}
		if err := s.profileRepo.Create(p); err != nil {
			return ProfileView{}, errInternal("не удалось создать профиль")
		}
		return ProfileView{Profile: *p, Online: false}, nil
	}
	if u.Username != "" {
		other, err := s.profileRepo.FindByUsername(u.Username)
		if err == nil && other.ID != p.ID {
			return ProfileView{}, errAlreadyExists("username taken")
		}
		p.Username = u.Username
	}
	if u.Email != "" && u.Email != p.Email {
		p.Email = u.Email
		p.EmailVerified = false
	}
	if u.Avatar != "" {
		p.Avatar = u.Avatar
	} else if u.ClearAvatar {
		p.Avatar = ""
	}
	if u.Bio != "" {
		p.Bio = u.Bio
	}
	if u.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", u.DateOfBirth); err == nil {
			p.DateOfBirth = &t
		}
	}
	if u.IsAdmin != nil {
		p.IsAdmin = *u.IsAdmin
	}
	if u.IsBot != nil {
		p.IsBot = *u.IsBot
	}
	if err := s.profileRepo.Update(p); err != nil {
		return ProfileView{}, errInternal("update failed")
	}
	return ProfileView{Profile: *p, Online: s.isOnline(ctx, p.ID)}, nil
}

func (s *Server) GetContacts(ctx context.Context, userID uint) ([]entity.Profile, error) {
	list, err := s.profileRepo.GetContacts(userID)
	if err != nil {
		return nil, errInternal("contacts failed")
	}
	for i := range list {
		s.hideAvatar(&list[i], userID)
	}
	return list, nil
}

func (s *Server) AddContact(ctx context.Context, userID, contactID uint) error {
	if err := s.profileRepo.AddContact(userID, contactID); err != nil {
		return errInvalid(err.Error())
	}
	return nil
}

func (s *Server) GetSettings(ctx context.Context, userID uint) (*entity.Profile, error) {
	p, err := s.profileRepo.FindByID(userID)
	if err != nil {
		return nil, errInternal("profile not found")
	}
	return p, nil
}

func (s *Server) UpdateSettings(ctx context.Context, userID uint, showOnline bool, lastSeenPrivacy, avatarPrivacy string) (*entity.Profile, error) {
	p, err := s.profileRepo.FindByID(userID)
	if err != nil {
		return nil, errInternal("profile not found")
	}
	p.ShowOnlineStatus = showOnline
	if lastSeenPrivacy != "" {
		p.LastSeenPrivacy = lastSeenPrivacy
	}
	if avatarPrivacy != "" {
		p.AvatarPrivacy = avatarPrivacy
	}
	if err := s.profileRepo.Update(p); err != nil {
		return nil, errInternal("update failed")
	}
	return p, nil
}

func (s *Server) ResolveUserByName(ctx context.Context, username string) (uint, bool, error) {
	p, err := s.profileRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return p.ID, true, nil
}

func (s *Server) hideAvatar(p *entity.Profile, viewerID uint) {
	if p == nil || viewerID == 0 || p.Avatar == "" {
		return
	}
	switch p.AvatarPrivacy {
	case "nobody":
		if p.ID != viewerID {
			p.Avatar = ""
		}
	case "contacts":
		if p.ID != viewerID {
			ok, err := s.profileRepo.IsContact(p.ID, viewerID)
			if err != nil || !ok {
				p.Avatar = ""
			}
		}
	}
}

func (s *Server) isOnline(ctx context.Context, userID uint) bool {
	return s.redis.IsOnline(ctx, userID)
}
