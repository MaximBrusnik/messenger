package service

import (
	"context"

	"messengermax/musicservice/internal/entity"
	"messengermax/musicservice/internal/integration/filestore"
	"messengermax/musicservice/internal/repo"
)

const maxStorage = 1 << 30 // 1 GB

type Server struct {
	musicRepo repo.MusicRepository
	store     *filestore.Store
}

func NewServer(musicRepo repo.MusicRepository, st *filestore.Store) *Server {
	return &Server{musicRepo: musicRepo, store: st}
}

func (s *Server) List(ctx context.Context) ([]entity.Music, error) {
	list, err := s.musicRepo.FindByStatus(entity.StatusApproved)
	if err != nil {
		return nil, errInternal("failed to list music")
	}
	return list, nil
}

func (s *Server) ListPending(ctx context.Context) ([]entity.Music, error) {
	list, err := s.musicRepo.FindByStatus(entity.StatusPending)
	if err != nil {
		return nil, errInternal("failed to list pending music")
	}
	return list, nil
}

func (s *Server) Create(ctx context.Context, m *entity.Music) error {
	total, err := s.musicRepo.GetTotalSize()
	if err != nil {
		return errInternal("failed to check storage")
	}
	if total+m.Size > maxStorage {
		return errResourceExhausted("лимит хранилища музыки достигнут")
	}
	if err := s.musicRepo.Create(m); err != nil {
		return errInternal("failed to save track")
	}
	return nil
}

func (s *Server) GetMusic(ctx context.Context, id uint) (*entity.Music, error) {
	m, err := s.musicRepo.FindByID(id)
	if err != nil {
		return nil, errNotFound("track not found")
	}
	return m, nil
}

func (s *Server) GetFilePath(ctx context.Context, id uint) (filePath, originalName string, err error) {
	m, err := s.musicRepo.FindByID(id)
	if err != nil {
		return "", "", errNotFound("track not found")
	}
	return s.store.Path(m.Filename), m.OriginalName, nil
}

func (s *Server) Approve(ctx context.Context, id uint) error {
	if err := s.musicRepo.UpdateStatus(id, entity.StatusApproved); err != nil {
		return errInternal("failed to approve")
	}
	return nil
}

func (s *Server) Reject(ctx context.Context, id uint) error {
	m, err := s.musicRepo.FindByID(id)
	if err != nil {
		return errNotFound("track not found")
	}
	if err := s.musicRepo.UpdateStatus(m.ID, entity.StatusRejected); err != nil {
		return errInternal("failed to reject")
	}
	s.store.Remove(m.Filename)
	return nil
}

func (s *Server) Delete(ctx context.Context, id uint) error {
	m, err := s.musicRepo.FindByID(id)
	if err != nil {
		return errNotFound("track not found")
	}
	if err := s.musicRepo.Delete(m.ID); err != nil {
		return errInternal("failed to delete")
	}
	s.store.Remove(m.Filename)
	return nil
}
