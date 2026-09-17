package service

import (
	"context"
	"strconv"
	"time"

	"messengermax/callsservice/internal/entity"
	"messengermax/callsservice/internal/repo"
	"messengermax/pkg/nats"
)

const RingTimeout = 30 * time.Second

type Server struct {
	callRepo repo.CallRepository
	producer *nats.Producer
}

func NewServer(callRepo repo.CallRepository, producer *nats.Producer) *Server {
	return &Server{callRepo: callRepo, producer: producer}
}

func (s *Server) StartCall(ctx context.Context, callerID, calleeID, chatID uint, callType entity.CallType) (*entity.Call, error) {
	if calleeID == 0 {
		return nil, errInvalid("callee required")
	}
	if callerID == calleeID {
		return nil, errInvalid("нельзя позвонить самому себе")
	}
	if _, err := s.callRepo.FindActiveForUser(callerID); err == nil {
		return nil, errFailedPrecondition("у вас уже есть активный звонок")
	}
	if _, err := s.callRepo.FindActiveForUser(calleeID); err == nil {
		return nil, errFailedPrecondition("собеседник уже в звонке")
	}
	now := time.Now().UnixMilli()
	call := &entity.Call{
		CallerID:    callerID,
		CalleeID:    calleeID,
		CallType:    callType,
		Status:      entity.CallStatusRinging,
		StartedAtMs: now,
		EndReason:   "timeout", // default for a ringing call that is never accepted
		TimeToDieAt: now + RingTimeout.Milliseconds(),
	}
	if err := s.callRepo.Create(call); err != nil {
		return nil, errInternal("не удалось создать звонок")
	}
	if chatID != 0 {
		call.ChatID = chatID
		_ = s.callRepo.Update(call)
	}
	return call, nil
}

func (s *Server) AcceptCall(ctx context.Context, callID, userID uint) (*entity.Call, error) {
	if callID == 0 {
		return nil, errInvalid("call_id required")
	}
	call, err := s.callRepo.FindByID(callID)
	if err != nil {
		return nil, errNotFound("звонок не найден")
	}
	if call.Status != entity.CallStatusRinging {
		return nil, errFailedPrecondition("звонок уже завершён")
	}
	call.Status = entity.CallStatusActive
	if userID != 0 {
		call.AcceptedAtMs = time.Now().UnixMilli()
		call.TimeToDieAt = 0
	}
	if err := s.callRepo.Update(call); err != nil {
		return nil, errInternal("не удалось обновить звонок")
	}
	return call, nil
}

func (s *Server) EndCall(ctx context.Context, callID, userID uint, reason string) (*entity.Call, error) {
	if callID == 0 {
		return nil, errInvalid("call_id required")
	}
	call, err := s.callRepo.FindByID(callID)
	if err != nil {
		return nil, errNotFound("звонок не найден")
	}
	if call.Status == entity.CallStatusEnded ||
		call.Status == entity.CallStatusCancelled ||
		call.Status == entity.CallStatusRejected ||
		call.Status == entity.CallStatusMissed {
		return call, nil
	}
	now := time.Now().UnixMilli()
	endedAt := now
	if call.AcceptedAtMs > 0 {
		call.DurationMs = now - call.AcceptedAtMs
	} else {
		call.DurationMs = 0
	}

	switch {
	case reason == "rejected":
		call.Status = entity.CallStatusRejected
	case reason == "no_answer" || reason == "unreachable" || reason == "timeout":
		call.Status = entity.CallStatusMissed
	case call.AcceptedAtMs == 0:
		call.Status = entity.CallStatusCancelled
	default:
		call.Status = entity.CallStatusEnded
	}
	call.EndedAtMs = endedAt
	call.EndReason = reason
	if err := s.callRepo.Update(call); err != nil {
		return nil, errInternal("не удалось завершить звонок")
	}
	s.publishCallEnded(call)
	return call, nil
}

func (s *Server) GetActiveCall(ctx context.Context, userID uint) (*entity.Call, error) {
	call, err := s.callRepo.FindActiveForUser(userID)
	if err != nil {
		return nil, errNotFound("нет активных звонков")
	}
	return call, nil
}

func (s *Server) GetCallHistory(ctx context.Context, userID uint, limit, offset int) ([]entity.Call, error) {
	calls, err := s.callRepo.History(userID, limit, offset)
	if err != nil {
		return nil, errInternal("не удалось получить историю звонков")
	}
	return calls, nil
}

// StartCallForSignaling is the entry point used by the WebSocket controller
// when a caller sends CALL_INVITE.
func (s *Server) StartCallForSignaling(ctx context.Context, callerID, calleeID uint, chatID uint, callType entity.CallType) (*entity.Call, error) {
	return s.StartCall(ctx, callerID, calleeID, chatID, callType)
}

func (s *Server) AcceptCallForSignaling(ctx context.Context, callID, userID uint) (*entity.Call, error) {
	return s.AcceptCall(ctx, callID, userID)
}

func (s *Server) EndCallForSignaling(ctx context.Context, callID, userID uint, reason string) (*entity.Call, error) {
	return s.EndCall(ctx, callID, userID, reason)
}

func (s *Server) FindActiveForSignaling(ctx context.Context, userID uint) (*entity.Call, error) {
	return s.GetActiveCall(ctx, userID)
}

func (s *Server) FindExpiredRinging(nowMs int64) ([]entity.Call, error) {
	return s.callRepo.FindExpiredRinging(nowMs)
}

func (s *Server) ExpireRingingCall(ctx context.Context, call *entity.Call) (*entity.Call, error) {
	return s.EndCall(ctx, call.ID, call.CallerID, "timeout")
}

func (s *Server) publishCallEnded(call *entity.Call) {
	if s.producer == nil || call.ChatID == 0 {
		return
	}
	s.producer.Publish(nats.TopicCallEnded, itoa(uint64(call.ID)), nats.EventCallEnded{
		CallID:      int64(call.ID),
		ChatID:      int64(call.ChatID),
		CallerID:    int64(call.CallerID),
		CalleeID:    int64(call.CalleeID),
		CallType:    string(call.CallType),
		Status:      string(call.Status),
		EndReason:   call.EndReason,
		StartedAtMs: call.StartedAtMs,
		EndedAtMs:   call.EndedAtMs,
		DurationMs:  call.DurationMs,
	})
}

func itoa(n uint64) string {
	return strconv.FormatUint(n, 10)
}
