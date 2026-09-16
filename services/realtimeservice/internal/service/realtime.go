package service

import (
	"context"

	"messengermax/pkg/redis"
	"messengermax/realtimeservice/internal/handler/ws"
)

type Server struct {
	redis  *redis.Client
	wsCtrl *ws.Controller
}

func NewServer(redis *redis.Client, wsCtrl *ws.Controller) *Server {
	return &Server{redis: redis, wsCtrl: wsCtrl}
}

func (s *Server) IsOnline(ctx context.Context, userID uint) (bool, error) {
	return s.redis.IsOnline(ctx, userID), nil
}

func (s *Server) GetOnlineStatuses(ctx context.Context, userIDs []uint) (map[uint64]bool, error) {
	out := make(map[uint64]bool, len(userIDs))
	for _, id := range userIDs {
		out[uint64(id)] = s.redis.IsOnline(ctx, id)
	}
	return out, nil
}

func (s *Server) Disconnect(ctx context.Context, userID uint) error {
	return s.wsCtrl.DisconnectUser(ctx, userID)
}
