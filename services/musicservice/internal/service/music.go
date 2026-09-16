package service

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/musicservice/internal/entity"
	"messengermax/musicservice/internal/repo"
	"messengermax/musicservice/internal/store"
	pb "messengermax/proto/gen/music"
)

const maxStorage = 1 << 30 // 1 GB

// Server implements the MusicService gRPC contract. Binary transfer happens
// separately over the HTTP file endpoints; this service owns metadata.
type Server struct {
	pb.UnimplementedMusicServiceServer
	musicRepo repo.MusicRepository
	store     *store.Store
}

func NewServer(musicRepo repo.MusicRepository, st *store.Store) *Server {
	return &Server{musicRepo: musicRepo, store: st}
}

func (s *Server) List(ctx context.Context, _ *pb.ListRequest) (*pb.MusicListResponse, error) {
	list, err := s.musicRepo.FindByStatus(entity.StatusApproved)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list music")
	}
	return &pb.MusicListResponse{Items: toProtoList(list)}, nil
}

func (s *Server) ListPending(ctx context.Context, _ *pb.ListRequest) (*pb.MusicListResponse, error) {
	list, err := s.musicRepo.FindByStatus(entity.StatusPending)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pending music")
	}
	return &pb.MusicListResponse{Items: toProtoList(list)}, nil
}

func (s *Server) Create(ctx context.Context, req *pb.CreateMusicRequest) (*pb.Music, error) {
	total, err := s.musicRepo.GetTotalSize()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check storage")
	}
	if total+req.Size > maxStorage {
		return nil, status.Error(codes.ResourceExhausted, "лимит хранилища музыки достигнут")
	}

	status_ := entity.StatusPending
	if req.IsAdmin {
		status_ = entity.StatusApproved
	}
	m := &entity.Music{
		Filename:     req.Filename,
		OriginalName: req.OriginalName,
		Title:        req.Title,
		Artist:       req.Artist,
		Size:         req.Size,
		MimeType:     req.MimeType,
		UploadedBy:   uint(req.UploadedBy),
		Status:       status_,
	}
	if err := s.musicRepo.Create(m); err != nil {
		return nil, status.Error(codes.Internal, "failed to save track")
	}
	return toProto(m), nil
}

func (s *Server) GetMusic(ctx context.Context, req *pb.GetMusicRequest) (*pb.Music, error) {
	m, err := s.musicRepo.FindByID(uint(req.Id))
	if err != nil {
		return nil, notFound(err)
	}
	return toProto(m), nil
}

func (s *Server) GetFilePath(ctx context.Context, req *pb.GetMusicRequest) (*pb.FilePathResponse, error) {
	m, err := s.musicRepo.FindByID(uint(req.Id))
	if err != nil {
		return nil, notFound(err)
	}
	return &pb.FilePathResponse{Path: s.store.Path(m.Filename), OriginalName: m.OriginalName}, nil
}

func (s *Server) Approve(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	if err := s.musicRepo.UpdateStatus(uint(req.Id), entity.StatusApproved); err != nil {
		return nil, status.Error(codes.Internal, "failed to approve")
	}
	return &pb.Empty{}, nil
}

func (s *Server) Reject(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	m, err := s.musicRepo.FindByID(uint(req.Id))
	if err != nil {
		return nil, notFound(err)
	}
	if err := s.musicRepo.UpdateStatus(m.ID, entity.StatusRejected); err != nil {
		return nil, status.Error(codes.Internal, "failed to reject")
	}
	s.store.Remove(m.Filename)
	return &pb.Empty{}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	m, err := s.musicRepo.FindByID(uint(req.Id))
	if err != nil {
		return nil, notFound(err)
	}
	if err := s.musicRepo.Delete(m.ID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete")
	}
	s.store.Remove(m.Filename)
	return &pb.Empty{}, nil
}

func notFound(err error) error {
	if errors.Is(err, repo.ErrNotFound) {
		return status.Error(codes.NotFound, "track not found")
	}
	return status.Error(codes.Internal, err.Error())
}

func toProto(m *entity.Music) *pb.Music {
	return &pb.Music{
		Id:           uint64(m.ID),
		Title:        m.Title,
		Artist:       m.Artist,
		OriginalName: m.OriginalName,
		Size:         m.Size,
		MimeType:     m.MimeType,
		UploadedBy:   uint64(m.UploadedBy),
		Status:       m.Status,
		CreatedAt:    timestamppb.New(m.CreatedAt),
	}
}

func toProtoList(list []entity.Music) []*pb.Music {
	out := make([]*pb.Music, 0, len(list))
	for i := range list {
		out = append(out, toProto(&list[i]))
	}
	return out
}
