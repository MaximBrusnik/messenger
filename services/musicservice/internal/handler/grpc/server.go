package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/musicservice/internal/entity"
	"messengermax/musicservice/internal/service"
	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/music"
)

type Server struct {
	pb.UnimplementedMusicServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) List(ctx context.Context, _ *pb.ListRequest) (*pb.MusicListResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.MusicListResponse{Items: toProtoList(list)}, nil
}

func (s *Server) ListPending(ctx context.Context, _ *pb.ListRequest) (*pb.MusicListResponse, error) {
	list, err := s.svc.ListPending(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.MusicListResponse{Items: toProtoList(list)}, nil
}

func (s *Server) Create(ctx context.Context, req *pb.CreateMusicRequest) (*pb.Music, error) {
	if req.Filename == "" || req.OriginalName == "" {
		return nil, status.Error(codes.InvalidArgument, "filename and original name required")
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
	if err := s.svc.Create(ctx, m); err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(m), nil
}

func (s *Server) GetMusic(ctx context.Context, req *pb.GetMusicRequest) (*pb.Music, error) {
	m, err := s.svc.GetMusic(ctx, uint(req.Id))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(m), nil
}

func (s *Server) GetFilePath(ctx context.Context, req *pb.GetMusicRequest) (*pb.FilePathResponse, error) {
	path, name, err := s.svc.GetFilePath(ctx, uint(req.Id))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.FilePathResponse{Path: path, OriginalName: name}, nil
}

func (s *Server) Approve(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	if err := s.svc.Approve(ctx, uint(req.Id)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) Reject(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	if err := s.svc.Reject(ctx, uint(req.Id)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.GetMusicRequest) (*pb.Empty, error) {
	if err := s.svc.Delete(ctx, uint(req.Id)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
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
